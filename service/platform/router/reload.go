package router

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/files"
	"github.com/viant/mly/service/platform"
	"github.com/viant/mly/shared/config/router"
	"gopkg.in/yaml.v2"
)

type modelSignature struct {
	name      string
	signature *domain.Signature
}

func (r *Router) ReloadIfNeeded(ctx context.Context) error {
	start := time.Now()
	isFullReload := false
	defer func() {
		var mode string
		if isFullReload {
			mode = "full"
		} else {
			mode = "checks"
		}
		routerReloadDurationMicrosSummary.WithLabelValues(r.routerName, mode).Observe(float64(time.Since(start).Microseconds()))
	}()

	// fetch and check router configuration file
	snapshot, err := files.ModifiedSnapshot(ctx, r.fs, r.configURL, nil)
	if err != nil {
		return fmt.Errorf("failed to check router configuration file: %w", err)
	}

	if !r.isModified(snapshot) {
		// check health of all underlying models
		var wg sync.WaitGroup

		r.configLock.RLock()
		errChannels := len(r.routingTable)
		if r.globalModel != nil {
			errChannels++
		}

		errCh := make(chan error, errChannels)

		if r.globalModel != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := r.globalModel.ReloadIfNeeded(ctx)
				if err != nil {
					errCh <- fmt.Errorf("failed to reload global model: %w", err)
				}
			}()
		}

		for m, p := range r.routingTable {
			wg.Add(1)
			go func(m string, p platform.PlatformEvaluator) {
				defer wg.Done()
				err := p.ReloadIfNeeded(ctx)
				if err != nil {
					errCh <- fmt.Errorf("failed to reload model %s: %w", m, err)
				}
			}(m, p)
		}

		wg.Wait()
		close(errCh)

		if len(errCh) > 0 {
			var errStrings []string
			for err := range errCh {
				errStrings = append(errStrings, err.Error())
			}

			err = fmt.Errorf("reloading errors: %s", strings.Join(errStrings, "; "))
		}

		r.configLock.RUnlock()
		return err
	}

	isFullReload = true

	// otherwise just abandon the routing table status checks

	r.configLock.Lock()
	defer r.configLock.Unlock()

	r.configModified = snapshot

	// load router configuration file
	rawReader, err := r.fs.OpenURL(ctx, r.configURL)
	if err != nil {
		return fmt.Errorf("failed to open router configuration file: %w", err)
	}

	defer rawReader.Close()
	var reader io.Reader = rawReader
	if strings.HasSuffix(r.configURL, ".gz") {
		if reader, err = gzip.NewReader(rawReader); err != nil {
			return fmt.Errorf("failed to create gzip reader for router configuration file: %w", err)
		}
	}

	newConfig := new(router.RoutingConfig)

	// TODO move this check earlier
	if strings.Contains(r.configURL, ".yaml") {
		decoder := yaml.NewDecoder(reader)
		err = decoder.Decode(newConfig)
	} else if strings.Contains(r.configURL, ".json") {
		err = json.NewDecoder(reader).Decode(newConfig)
	} else {
		return fmt.Errorf("unsupported router configuration file type: %s", r.configURL)
	}

	if err != nil {
		return fmt.Errorf("failed to decode router configuration file: %w", err)
	}

	if err := r.applyRouterConfig(ctx, newConfig); err != nil {
		return err
	}

	return nil
}

// applyRouterConfig will both update evaluators to new configuration state and verify and build the signature
func (r *Router) applyRouterConfig(ctx context.Context, newConfig *router.RoutingConfig) error {
	modelsToUnload := make(map[string]struct{})
	reuseEvaluators := make(map[string]platform.PlatformEvaluator)
	var reuseGlobal platform.PlatformEvaluator

	var finalSignature *domain.Signature
	var oldConfig *router.RoutingConfig
	func() {
		r.routingTableLock.RLock()
		defer r.routingTableLock.RUnlock()
		if r.ioState != nil {
			finalSignature = r.ioState.signature
		}

		reuseGlobal = r.globalModel
		oldConfig = r.routingConfig
	}()

	if oldConfig != nil {
		for _, entity := range oldConfig.EntityMapping {
			modelsToUnload[entity.ModelName] = struct{}{}
			if evaluator, ok := r.routingTable[entity.ModelName]; ok {
				reuseEvaluators[entity.ModelName] = evaluator
			}
		}

		if oldConfig.GlobalModelName != "" {
			modelsToUnload[oldConfig.GlobalModelName] = struct{}{}
		}
	}

	newModelMapping := make(map[int]string)
	for _, entity := range newConfig.EntityMapping {
		r.debugLogf("add mapping: %d -> %s", entity.EntityID, entity.ModelName)

		newModelMapping[entity.EntityID] = entity.ModelName
		delete(modelsToUnload, entity.ModelName)
	}

	globalModelName := newConfig.GlobalModelName
	if globalModelName == "" && r.hasGlobalModel {
		return fmt.Errorf("global model name is missing")
	}

	if globalModelName != "" {
		r.debugLogf("global model: %s", globalModelName)
		delete(modelsToUnload, globalModelName)
	}

	newRoutingTable := make(map[string]platform.PlatformEvaluator)
	for _, entity := range newConfig.EntityMapping {
		model := entity.ModelName
		if _, ok := newRoutingTable[model]; ok {
			continue
		}

		if evaluator, ok := reuseEvaluators[model]; ok {
			newRoutingTable[model] = evaluator
			continue
		}

		evaluator, err := r.makeRoutedEvaluator(model)

		if err != nil {
			return fmt.Errorf("failed to create Routed Evaluator for model %s: %w", model, err)
		}

		newRoutingTable[model] = evaluator
	}

	var globalEvaluator platform.PlatformEvaluator
	if globalModelName != "" {
		if oldConfig != nil && globalModelName == oldConfig.GlobalModelName && reuseGlobal != nil {
			globalEvaluator = reuseGlobal
		} else if evaluator, ok := newRoutingTable[globalModelName]; ok {
			globalEvaluator = evaluator
		} else if evaluator, ok := reuseEvaluators[globalModelName]; ok {
			globalEvaluator = evaluator
		} else {
			var err error
			globalEvaluator, err = r.makeRoutedEvaluator(globalModelName)
			if err != nil {
				return fmt.Errorf("failed to create Routed Evaluator for global model %s: %w", globalModelName, err)
			}
		}
	}

	wg := sync.WaitGroup{}

	numWorkers := len(newRoutingTable)
	if globalEvaluator != nil {
		numWorkers++
	}

	errCh := make(chan error, numWorkers)
	signatureCh := make(chan modelSignature, numWorkers)

	if globalEvaluator != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := globalEvaluator.ReloadIfNeeded(ctx); err != nil {
				errCh <- fmt.Errorf("failed to reload global model %s: %w", globalModelName, err)
			}
		}()
	}

	for model := range newRoutingTable {
		wg.Add(1)
		go func(model string) {
			defer wg.Done()

			r.debugLogf("reload model: %s", model)

			modelEvaluator := newRoutingTable[model]
			if err := modelEvaluator.ReloadIfNeeded(ctx); err != nil {
				r.debugLogf("failed to reload model: %s: %v", model, err)
				errCh <- fmt.Errorf("failed to reload model %s: %w", model, err)
			}

			evalSig := modelEvaluator.Signature()

			if evalSig == nil {
				errCh <- fmt.Errorf("model %s signature is nil", model)
				return
			}

			signatureCh <- modelSignature{
				name:      model,
				signature: evalSig,
			}
		}(model)
	}

	r.debugLogf("wait for reloads")

	wg.Wait()
	close(errCh)
	close(signatureCh)

	if len(errCh) > 0 {
		var errStrings []string
		for err := range errCh {
			errStrings = append(errStrings, err.Error())
		}
		return fmt.Errorf("one or more model reloading errors: %s", strings.Join(errStrings, "; "))
	}

	sigInputMap := make(map[string]*domain.Input)
	sigOutputMap := make(map[string]*domain.Output)

	// we only create ioState on the first reload
	var ioState *IOState = new(IOState)
	if finalSignature != nil {
		for _, input := range finalSignature.Inputs {
			sigInputMap[input.Name] = &input
		}

		for _, output := range finalSignature.Outputs {
			sigOutputMap[output.Name] = &output
		}
	}

	for signature := range signatureCh {
		// accept first available signature as the final signature
		if finalSignature == nil {
			// in the creation of the signature, include the routing input
			// DANGER: this uses the pointer to the signature, so since the signature is modified, the original signature will be modified!
			// This doesn't happen in practice, but can cause issues in tests.
			finalSignature = signature.signature

			inputOffset := len(finalSignature.Inputs)

			routerInput := domain.Input{
				Name: r.routerInputFieldName,
				Type: reflect.TypeOf(int64(0)),
			}

			ioState.routerInputOffset = inputOffset

			finalSignature.Inputs = append(finalSignature.Inputs, routerInput)

			for _, input := range finalSignature.Inputs {
				sigInputMap[input.Name] = &input
			}

			for _, input := range r.configuredInputs {
				_, ok := sigInputMap[input.Name]

				if ok {
					// the input is configured and already in the self-reported signature
					continue
				}

				if !input.Auxiliary {
					return fmt.Errorf("non-auxiliary input %s for model %s was not in model inputs", input.Name, signature.name)
				}

				sigInputMap[input.Name] = &domain.Input{
					Name:      input.Name,
					Type:      input.RawType(),
					Auxiliary: input.Auxiliary,
				}
			}

			if r.modelOutputName != "" {
				// also, add the selected model output
				modelOutput := domain.Output{
					Name:     r.modelOutputName,
					Index:    len(finalSignature.Outputs),
					DataType: "string",
				}

				finalSignature.Outputs = append(finalSignature.Outputs, modelOutput)
			}

			for _, output := range finalSignature.Outputs {
				sigOutputMap[output.Name] = &output
			}

			continue
		}

		thisSignature := signature.signature
		// validate signature consistency
		thisSignatureOutputMap := make(map[string]*domain.Output)
		for _, output := range thisSignature.Outputs {
			oldOutput, ok := sigOutputMap[output.Name]
			if !ok {
				return fmt.Errorf("signature output %s for model %s not found in the previous signature", output.Name, signature.name)
			}

			thisSignatureOutputMap[output.Name] = &output

			// TODO permit this
			if oldOutput.Index != output.Index {
				return fmt.Errorf("signature output %s for model %s has index %d, and the previous signature has index %d", output.Name, signature.name, output.Index, oldOutput.Index)
			}

			if oldOutput.DataType != output.DataType {
				return fmt.Errorf("signature output %s for model %s has data type %s, and the previous signature has data type %s", output.Name, signature.name, output.DataType, oldOutput.DataType)
			}
		}

		for expectedOutput := range sigOutputMap {
			if _, ok := thisSignatureOutputMap[expectedOutput]; !ok && expectedOutput != r.modelOutputName {
				return fmt.Errorf("signature output %s for was not found in model %s signature", expectedOutput, signature.name)
			}
		}

		thisSignatureInputMap := make(map[string]*domain.Input)
		for _, input := range thisSignature.Inputs {
			oldInput, ok := sigInputMap[input.Name]
			if !ok {
				return fmt.Errorf("signature input %s for model %s not found in the previous signature", input.Name, signature.name)
			}

			thisSignatureInputMap[input.Name] = &input

			if oldInput.Auxiliary {
				continue
			}

			// TODO permit this
			if oldInput.Index != input.Index {
				return fmt.Errorf("signature input %s for model %s has index %d, and the previous signature has index %d", input.Name, signature.name, input.Index, oldInput.Index)
			}

			if !oldInput.Type.ConvertibleTo(input.Type) {
				return fmt.Errorf("signature input %s for model %s has data type %s, and the previous signature has data type %s", input.Name, signature.name, input.Type.String(), oldInput.Type.String())
			}
		}

		for expectedInput := range sigInputMap {
			if _, ok := thisSignatureInputMap[expectedInput]; !ok && expectedInput != r.routerInputFieldName {
				return fmt.Errorf("signature input %s for was not found in model %s signature", expectedInput, signature.name)
			}
		}
	}

	if r.fixedEvaluatorFields != nil {
		// TODO this is actually an acceptable case, but needs to be addressed elsewhere first before it is permitted
		for field := range r.fixedEvaluatorFields {
			if _, ok := sigOutputMap[field]; !ok {
				return fmt.Errorf("fixed evaluator field: %s was not found in the signature outputs", field)
			}
		}

		for _, field := range sigOutputMap {
			if _, ok := r.fixedEvaluatorFields[field.Name]; !ok && field.Name != r.modelOutputName {
				return fmt.Errorf("signature output %s is not replaced", field.Name)
			}
		}
	}

	ioState.signature = finalSignature
	ioState.inputs = sigInputMap

	if globalEvaluator != nil {
		if _, exists := newRoutingTable[globalModelName]; !exists {
			newRoutingTable[globalModelName] = globalEvaluator
		}
	}

	func() {
		r.routingTableLock.Lock()
		defer r.routingTableLock.Unlock()

		r.routingConfig = newConfig

		r.routingMap = newModelMapping
		r.routingTable = newRoutingTable

		r.globalModel = globalEvaluator

		if r.ioState == nil {
			r.ioState = ioState
		}
	}()

	for model := range modelsToUnload {
		routerModelUnloadGauge.WithLabelValues(r.routerName).Inc()

		go func(modelName string) {
			defer routerModelUnloadGauge.WithLabelValues(r.routerName).Dec()

			ctxTo, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			r.debugLogf("request to unload model: %s", modelName)

			if err := r.unloadModel(ctxTo, modelName); err != nil {
				r.debugLogf("failed to unload model %s: %v\n", modelName, err)
			}
		}(model)
	}

	return nil
}

func (r *Router) debugLogf(format string, args ...interface{}) {
	if r.debug {
		prefix := "[%s Router] "
		log.Printf(prefix+format, append([]interface{}{r.routerName}, args...)...)
	}
}

// TODO refactor with service/tfmodel/service.isModified()?
func (r *Router) isModified(snapshot *config.Modified) bool {
	if r.routingConfig == nil || r.configModified == nil {
		return true
	}

	if snapshot.Max.IsZero() {
		return false
	}

	r.configLock.RLock()
	modified := r.configModified
	r.configLock.RUnlock()

	return !(modified.Max.Equal(snapshot.Max) && modified.Min.Equal(snapshot.Min))
}

func (r *Router) unloadModel(ctx context.Context, modelName string) error {
	if err := r.unloader.UnloadModel(ctx, r.routerName, modelName); err != nil {
		return fmt.Errorf("failed to unload model %s: %w", modelName, err)
	}
	return nil
}
