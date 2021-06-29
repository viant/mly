package endpoint

//Options represents an option
type Options struct {
	ConfigURL string `short:"c" long:"cfg" description:"config URI"`
	Version   bool   `short:"v" long:"version" description:"indexer version"`
}
