package logx

type Config struct {
	Level      string `mapstructure:"level" json:"level"`
	Target     string `mapstructure:"target" json:"target"`
	Format     string `mapstructure:"format" json:"format"`
	FilePath   string `mapstructure:"file_path" json:"file_path"`
	MaxBackups int    `mapstructure:"max_backups" json:"max_backups"`
	MaxSize    int    `mapstructure:"max_size" json:"max_size"`
}
