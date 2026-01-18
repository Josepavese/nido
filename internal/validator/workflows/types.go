package workflows

// Definition is the root workflow configuration.
type Definition struct {
	Workflows map[string]Workflow `yaml:"workflows"`
}

// Workflow describes a sequence of actions.
type Workflow struct {
	Steps []Step `yaml:"steps"`
}

// Step is a generic action used by both CLI and MCP runners.
type Step struct {
	Action          string `yaml:"action"`           // spawn, template_create, template_delete, delete_vm, image_pull, cache_rm
	VMVar           string `yaml:"vm_var,omitempty"` // variable name to store/reuse VM name
	TemplateVar     string `yaml:"template_var,omitempty"`
	UseBaseTemplate bool   `yaml:"use_base_template,omitempty"`
	UseBaseImage    bool   `yaml:"use_base_image,omitempty"`
	Image           string `yaml:"image,omitempty"` // explicit image to use/pull
	ExpectCacheHit  bool   `yaml:"expect_cache_hit,omitempty"`
}
