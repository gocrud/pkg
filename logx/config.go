package logx

type Config struct {
	Level      string `json:"level"`
	Target     string `json:"target"`
	Format     string `json:"format"`
	FilePath   string `json:"file_path"`
	MaxBackups int    `json:"max_backups"`
	MaxSize    int    `json:"max_size"`
}
