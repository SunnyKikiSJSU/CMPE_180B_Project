package database

import (
	"context"
	"errors"
	"strings"
	"testing"

	"book-metaverse/backend/internal/config"
)

type fakeSecrets struct {
	value string
	err   error
}

func (f fakeSecrets) GetSecretString(context.Context, string) (string, error) { return f.value, f.err }

func secretConfig() *config.Config {
	return &config.Config{DatabaseSource: config.SourceSecret, DBSecretARN: "arn:aws:secretsmanager:us-west-2:1:secret:x", DBSSLMode: "require"}
}

func TestResolveURLFromTerraformSecret(t *testing.T) {
	// Same JSON shape as infra/secrets.tf (port is a number).
	secret := `{"engine":"postgres","host":"db.example.us-west-2.rds.amazonaws.com","port":5432,"dbname":"appdb","username":"appadmin","password":"p/ss:w@rd"}`
	url, err := ResolveURL(context.Background(), secretConfig(), fakeSecrets{value: secret})
	if err != nil {
		t.Fatal(err)
	}
	want := "postgres://appadmin:p%2Fss%3Aw%40rd@db.example.us-west-2.rds.amazonaws.com:5432/appdb?sslmode=require"
	if url != want {
		t.Fatalf("got %s", url)
	}
}

func TestResolveURLErrors(t *testing.T) {
	cases := map[string]fakeSecrets{
		"AWS error":      {err: errors.New("AccessDenied")},
		"not JSON":       {value: "nope"},
		"missing fields": {value: `{"host":"h","port":5432}`},
		"bad port":       {value: `{"host":"h","port":"x","dbname":"d","username":"u","password":"p"}`},
	}
	for name, getter := range cases {
		_, err := ResolveURL(context.Background(), secretConfig(), getter)
		if err == nil {
			t.Errorf("%s: expected an error", name)
		} else if strings.Contains(err.Error(), "password") && strings.Contains(err.Error(), `"p"`) {
			t.Errorf("%s: error leaks the secret: %v", name, err)
		}
	}
}

func TestResolveURLOtherSources(t *testing.T) {
	cfg := &config.Config{DatabaseSource: config.SourceEnv, DatabaseURL: "postgres://a@b/c"}
	url, err := ResolveURL(context.Background(), cfg, fakeSecrets{err: errors.New("must not be called")})
	if err != nil || url != cfg.DatabaseURL {
		t.Fatalf("got %q, %v", url, err)
	}
}
