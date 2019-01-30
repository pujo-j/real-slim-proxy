package proxy

import (
	"context"
	"fmt"
	"gocloud.dev/blob"
	"gocloud.dev/blob/gcsblob"
	"gocloud.dev/gcp"
)

type StoreConfig struct {
	Cloud  string
	Bucket string
}

func (s *StoreConfig) OpenStore(ctx context.Context) (*blob.Bucket, error) {
	switch s.Cloud {
	case "gcp":
		creds, err := gcp.DefaultCredentials(ctx)
		if err != nil {
			return nil, err
		}
		c, err := gcp.NewHTTPClient(gcp.DefaultTransport(), gcp.CredentialsTokenSource(creds))
		if err != nil {
			return nil, err
		}
		return gcsblob.OpenBucket(ctx, c, s.Bucket, nil)
	case "aws", "azure":
		//TODO: ADD AWS and Azure storage
		return nil, fmt.Errorf("unimplemented cloud type : %v", s.Cloud)
	default:
		return nil, fmt.Errorf("unknown cloud type : %v", s.Cloud)
	}
}
