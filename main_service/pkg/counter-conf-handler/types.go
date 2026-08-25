package counterconfhandler

//All used for reading the counter-monitor-conf.yaml file

type Counters struct {
	Name string
	Path string
}

type Config struct {
	Interval int
	Counters []Counters
}

type Counter_conf struct {
	Config []Config
}
