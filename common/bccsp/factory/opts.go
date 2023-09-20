package factory

// GetDefaultOpts offers a default implementation for Opts
// returns a new instance every time
func GetDefaultOpts() *FactoryOpts {
	return &FactoryOpts{
		ProviderName: "SW",
		SwOpts: &SwOpts{
			HashFamily: "SHA2",
			SecLevel:   256,

			Ephemeral: true,
		},
	}
}

// FactoryName returns the name of the provider
func (o *FactoryOpts) FactoryName() string {
	return o.ProviderName
}
