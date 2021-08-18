package bitbucket

type Descriptor struct {
	Key            string                    `json:"key,omitempty"`
	Name           string                    `json:"name,omitempty"`
	Description    string                    `json:"description,omitempty"`
	Vendor         DescriptorVendor          `json:"vendor,omitempty"`
	BaseURL        string                    `json:"baseUrl,omitempty"`
	Authentication *DescriptorAuthentication `json:"authentication,omitempty"`
	Lifecycle      *DescriptorLifecycle      `json:"lifecycle,omitempty"`
	Scopes         []string                  `json:"scopes,omitempty"`
	Contexts       []string                  `json:"contexts,omitempty"`
	Modules        *DescriptorModules        `json:"modules,omitempty"`
}

type DescriptorVendor struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type DescriptorAuthentication struct {
	Type string `json:"type,omitempty"`
}

type DescriptorLifecycle struct {
	Installed   string `json:"installed,omitempty"`
	Uninstalled string `json:"uninstalled,omitempty"`
}

type DescriptorModules struct {
	Webhooks []DescriptorWebhook `json:"webhooks,omitempty"`
}

type DescriptorWebhook struct {
	Event string `json:"event,omitempty"`
	URL   string `json:"url,omitempty"`
}
