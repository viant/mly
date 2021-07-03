package lt

//Options represents load testing options
type Options struct {
	ConfigURL string `short:"c" long:"config" description:"config url"`
	DataFile  string `short:"d" long:"data" description:"data location"`
	Repeat    int    `short:"r" long:"repeat" description:"repeat"`
}
