package app

type Config struct {
	Gateway struct {
		Address string `yaml:"Address"`
	} `yaml:"Gateway"`
	Storage struct {
		Endpoint     string `yaml:"Endpoint"`
		AccessKey    string `yaml:"AccessKey"`
		AccessSecret string `yaml:"AccessSecret"`
		Region       string `yaml:"Region"`
		Bucket       string `yaml:"Bucket"`
	} `yaml:"Storage"`
}
