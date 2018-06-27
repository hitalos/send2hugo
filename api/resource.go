package api

type resource struct {
	Src    string            `json:"src,omitempty" yaml:"src,omitempty"`
	Title  string            `json:"title,omitempty" yaml:"title,omitempty"`
	Params map[string]string `json:"params,omitempty" yaml:"params,omitempty"`
}
