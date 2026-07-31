package router

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"reflect"
	"sort"
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

	// see defer above
	isFullReload = true

	// otherwise just abandon the routing table status checks

	r.configLock.Lock()
	defer r.configLock.Unlock()

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

	// Record the snapshot only after the configuration has been fully applied.
	// Recording it before applyRouterConfig would mark a failed reload as the
	// current state, so isModified() would report no change and the new config
	// would never be retried until the file changed again.
	r.configModified = snapshot

	return nil
}

// applyRouterConfig will both update evaluators to new configuration state and verify and build the signature
func (r *Router) applyRouterConfig(ctx context.Context, newConfig *router.RoutingConfig) error {
	modelsToUnload := make(map[string]struct{})
	reuseEvaluators := make(map[string]platform.PlatformEvaluator)
	var reuseGlobal platform.PlatformEvaluator

	// copy members to local scope
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
		// The global model is a served model, so its signature always takes part
		// in building and validating the router IO. This also covers a global-only
		// config (empty entityMapping): without it the router would have no
		// signature and fail on cold start (nil signature, or a fixed-evaluator
		// field check against no collected outputs).
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := globalEvaluator.ReloadIfNeeded(ctx); err != nil {
				errCh <- fmt.Errorf("failed to reload global model %s: %w", globalModelName, err)
				return
			}

			evalSig := globalEvaluator.Signature()
			if evalSig == nil {
				errCh <- fmt.Errorf("global model %s signature is nil", globalModelName)
				return
			}

			signatureCh <- modelSignature{
				name:      globalModelName,
				signature: evalSig,
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
				return
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

	// sigOutputMap is for validating output consistency
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

	// dsmi stands for DownStream Model Information
	for dsmi := range signatureCh {
		// accept first available signature as the final signature
		if finalSignature == nil {
			srcSig := dsmi.signature

			finalOutputs, err := r.buildFinalOutputs(srcSig)
			if err != nil {
				return fmt.Errorf("failed to build router output signature for model %s: %w", dsmi.name, err)
			}

			// copy signature from downstream
			finalSignature = &domain.Signature{
				Inputs:  make([]domain.Input, len(srcSig.Inputs), len(srcSig.Inputs)+1),
				Outputs: finalOutputs,
			}
			copy(finalSignature.Inputs, srcSig.Inputs)

			// add router input
			inputOffset := len(finalSignature.Inputs)
			routerInput := domain.Input{
				Name:  r.routerInputFieldName,
				Type:  reflect.TypeOf(int64(0)),
				Index: inputOffset,
			}

			ioState.routerInputOffset = inputOffset

			finalSignature.Inputs = append(finalSignature.Inputs, routerInput)

			// sigInputMap is for Request validation
			for _, input := range finalSignature.Inputs {
				sigInputMap[input.Name] = &input
			}

			// add configured (aux) inputs to signature
			for _, input := range r.configuredInputs {
				_, ok := sigInputMap[input.Name]

				if ok {
					// the input is configured and already in the self-reported signature
					continue
				}

				if !input.Auxiliary {
					return fmt.Errorf("non-auxiliary input %s for model %s was not in model inputs", input.Name, dsmi.name)
				}

				sigInputMap[input.Name] = &domain.Input{
					Name:      input.Name,
					Type:      input.RawType(),
					Auxiliary: input.Auxiliary,
				}
			}

			for _, output := range finalSignature.Outputs {
				sigOutputMap[output.Name] = &output
			}

			continue
		}

		dsSignature := dsmi.signature
		// validate signature consistency
		// Note: Index differences are permitted - IOs are matched by name

		// check that the new signature has no new outputs
		thisSignatureOutputMap := make(map[string]*domain.Output)
		for _, output := range dsSignature.Outputs {
			oldOutput, ok := sigOutputMap[output.Name]
			if !ok {
				return fmt.Errorf("signature output %s for model %s not found in the previous signature", output.Name, dsmi.name)
			}

			thisSignatureOutputMap[output.Name] = &output

			if oldOutput.DataType != output.DataType {
				return fmt.Errorf("signature output %s for model %s has data type %s, and the previous signature has data type %s", output.Name, dsmi.name, output.DataType, oldOutput.DataType)
			}
		}

		// check that the new signature has no new outputs except the model name output
		for expectedOutput := range sigOutputMap {
			if _, ok := thisSignatureOutputMap[expectedOutput]; !ok && expectedOutput != r.modelOutputName {
				return fmt.Errorf("signature output %s for was not found in model %s signature", expectedOutput, dsmi.name)
			}
		}

		// check that the new signature has no new inputs
		thisSignatureInputMap := make(map[string]*domain.Input)
		for _, input := range dsSignature.Inputs {
			oldInput, ok := sigInputMap[input.Name]
			if !ok {
				return fmt.Errorf("signature input %s for model %s not found in the previous signature", input.Name, dsmi.name)
			}

			thisSignatureInputMap[input.Name] = &input

			if !oldInput.Type.ConvertibleTo(input.Type) {
				return fmt.Errorf("signature input %s for model %s has data type %s, and the previous signature has data type %s", input.Name, dsmi.name, input.Type.String(), oldInput.Type.String())
			}
		}

		// check that the new signature has all expected inputs except for the routing and auxiliary inputs
		for expectedInput := range sigInputMap {
			if sigInputMap[expectedInput].Auxiliary {
				continue
			}

			if expectedInput == r.routerInputFieldName {
				continue
			}

			if _, ok := thisSignatureInputMap[expectedInput]; !ok {
				return fmt.Errorf("signature input %s for was not found in model %s signature", expectedInput, dsmi.name)
			}
		}
	}

	if r.fixedEvaluatorFields != nil {
		// TODO this is actually an acceptable case, we can simply ignore fixed evaluator fields that aren't applicable
		for field := range r.fixedEvaluatorFields {
			if _, ok := sigOutputMap[field]; !ok {
				return fmt.Errorf("fixed evaluator field: %s was not found in any model outputs", field)
			}
		}

		// check that the fixed evaluator fields have all expected outputs
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
		r.unloadGauge.Inc()

		go func(modelName string) {
			defer r.unloadGauge.Dec()

			ctxTo, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

func (r *Router) buildFinalOutputs(srcSig *domain.Signature) ([]domain.Output, error) {
	if len(r.configuredOutputs) == 0 {
		outputs := make([]domain.Output, len(srcSig.Outputs), len(srcSig.Outputs)+1)
		copy(outputs, srcSig.Outputs)
		if r.modelOutputName != "" {
			outputs = append(outputs, r.routerModelOutput(len(outputs)))
		}
		return outputs, nil
	}

	outputsByName := make(map[string]domain.Output, len(srcSig.Outputs))
	unconfiguredOutputs := make(map[string]struct{}, len(srcSig.Outputs))
	for _, output := range srcSig.Outputs {
		outputsByName[output.Name] = output
		unconfiguredOutputs[output.Name] = struct{}{}
	}

	outputs := make([]domain.Output, 0, len(r.configuredOutputs)+1)
	hasModelOutput := false
	for _, configured := range r.configuredOutputs {
		if configured.Name == r.modelOutputName && r.modelOutputName != "" {
			outputs = append(outputs, r.routerModelOutput(len(outputs)))
			hasModelOutput = true
			continue
		}

		output, ok := outputsByName[configured.Name]
		if !ok {
			return nil, fmt.Errorf("configured output %s was not found in model outputs", configured.Name)
		}

		output.Index = len(outputs)
		outputs = append(outputs, output)
		delete(unconfiguredOutputs, output.Name)
	}

	if len(unconfiguredOutputs) > 0 {
		names := make([]string, 0, len(unconfiguredOutputs))
		for name := range unconfiguredOutputs {
			names = append(names, name)
		}
		sort.Strings(names)
		return nil, fmt.Errorf("model outputs not present in configured outputs: %s", strings.Join(names, ", "))
	}

	if r.modelOutputName != "" && !hasModelOutput {
		outputs = append(outputs, r.routerModelOutput(len(outputs)))
	}

	return outputs, nil
}

func (r *Router) routerModelOutput(index int) domain.Output {
	return domain.Output{
		Name:     r.modelOutputName,
		Index:    index,
		DataType: "string",
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
