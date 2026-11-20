package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"book-metaverse/backend/internal/config"
)

// SecretGetter returns a secret's SecretString. Tests use a fake.
type SecretGetter interface {
	GetSecretString(ctx context.Context, arn string) (string, error)
}

// dbSecret is the JSON that infra/secrets.tf stores.
type dbSecret struct {
	Host     string          `json:"host"`
	Port     json.RawMessage `json:"port"`
	DBName   string          `json:"dbname"`
	Username string          `json:"username"`
	Password string          `json:"password"`
}

// ResolveURL returns the connection URL for cfg, reading the Secrets Manager
// secret when cfg.DatabaseSource is SourceSecret.
func ResolveURL(ctx context.Context, cfg *config.Config, getter SecretGetter) (string, error) {
	if cfg.DatabaseSource != config.SourceSecret {
		return cfg.DatabaseURL, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	raw, err := getter.GetSecretString(ctx, cfg.DBSecretARN)
	if err != nil {
		return "", fmt.Errorf("read database secret: %w", err)
	}
	var secret dbSecret
	if err := json.Unmarshal([]byte(raw), &secret); err != nil {
		return "", errors.New("read database secret: secret is not the expected JSON")
	}
	port, err := secretPort(secret.Port)
	if err != nil {
		return "", err
	}
	if secret.Host == "" || secret.DBName == "" || secret.Username == "" || secret.Password == "" {
		return "", errors.New("read database secret: host, dbname, username, and password are required")
	}
	return config.BuildURL(secret.Host, port, secret.DBName, secret.Username, secret.Password, cfg.DBSSLMode), nil
}

// The Terraform secret stores port as a number; accept a string too.
func secretPort(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "5432", nil
	}
	var number int
	if err := json.Unmarshal(raw, &number); err == nil && number > 0 && number < 65536 {
		return strconv.Itoa(number), nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if n, err := strconv.Atoi(text); err == nil && n > 0 && n < 65536 {
			return text, nil
		}
	}
	return "", errors.New("read database secret: port is invalid")
}

// AWSSecrets reads secrets with the default AWS credential chain (on EC2,
// the instance profile from infra/iam.tf).
type AWSSecrets struct{ client *secretsmanager.Client }

func NewAWSSecrets(ctx context.Context, region string) (*AWSSecrets, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	return &AWSSecrets{client: secretsmanager.NewFromConfig(awsCfg)}, nil
}

func (a *AWSSecrets) GetSecretString(ctx context.Context, arn string) (string, error) {
	out, err := a.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(arn)})
	if err != nil {
		return "", err
	}
	if out.SecretString == nil {
		return "", errors.New("secret has no SecretString")
	}
	return *out.SecretString, nil
}
