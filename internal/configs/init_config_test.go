package configs

import (
	"os"
	"testing"
)

// InitConfig is the difference between a server that starts against the wrong database and one that
// does not start at all, so the path it takes on every boot is worth pinning down: the file is read,
// and MYAPP_* wins over it.
//
// The failure paths call log.Fatalf, which would take the test binary down with them - so they are
// deliberately not tested here, and that is worth knowing about this package: a bad config is a
// process exit, not an error a caller can handle.

const configFile = `
redis:
  url: redis://localhost:6379
jwt:
  secret: from-the-file
database:
  url: mongodb://localhost:27017
  name: game-center
google:
  client_id: google-from-file
apple:
  client_id: apple-from-file
  google_client_id: apple-google-from-file
`

func writeConfig(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.WriteFile("config.yaml", []byte(configFile), 0o600); err != nil {
		t.Fatalf("writing config.yaml: %v", err)
	}
}

func TestInitConfig_ReadsEverySection(t *testing.T) {
	writeConfig(t)
	InitConfig()

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"redis.url", ServerConfig.Redis.Url, "redis://localhost:6379"},
		{"jwt.secret", ServerConfig.Jwt.Secret, "from-the-file"},
		{"database.url", ServerConfig.Database.Url, "mongodb://localhost:27017"},
		{"database.name", ServerConfig.Database.Name, "game-center"},
		{"google.client_id", ServerConfig.Google.ClientId, "google-from-file"},
		{"apple.client_id", ServerConfig.Apple.ClientId, "apple-from-file"},
		{"apple.google_client_id", ServerConfig.Apple.GoogleClientId, "apple-google-from-file"},
	}

	for _, test := range tests {
		if test.got != test.want {
			t.Errorf("%s = %q, want %q", test.name, test.got, test.want)
		}
	}
}

// The env prefix exists so a container can be pointed at another database without rebuilding the
// image, which only works if the environment really does win over the file it ships with.
func TestInitConfig_LetsTheEnvironmentWin(t *testing.T) {
	writeConfig(t)
	t.Setenv("MYAPP_DATABASE_NAME", "from-the-environment")
	t.Setenv("MYAPP_JWT_SECRET", "from-the-environment")

	InitConfig()

	if ServerConfig.Database.Name != "from-the-environment" {
		t.Errorf("database.name = %q, want the environment to win", ServerConfig.Database.Name)
	}
	if ServerConfig.Jwt.Secret != "from-the-environment" {
		t.Errorf("jwt.secret = %q, want the environment to win", ServerConfig.Jwt.Secret)
	}
}
