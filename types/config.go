package types

type (
	Configure struct {
		Log   Log            `mapstructure:"log"`
		Watch Watch          `mapstructure:"watch"`
		Build Build          `mapstructure:"build"`
		Run   map[string]Run `mapstructure:"run"`
	}

	Run struct {
		Command []string `mapstructure:"command"`
		Only    bool     `mapstructure:"only"`
	}

	Build struct {
		Delay   int      `mapstructure:"delay"`
		Name    string   `mapstructure:"name"`
		Package string   `mapstructure:"package"`
		Command []string `mapstructure:"command"`
	}

	Watch struct {
		Include []string `mapstructure:"include"`
		Exclude []string `mapstructure:"exclude"`
	}

	Log struct {
		Level int    `mapstructure:"level"`
		Lang  string `mapstructure:"lang"`
	}
)
