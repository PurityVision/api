package src

import (
	"context"

	vision "cloud.google.com/go/vision/apiv1"
	pb "google.golang.org/genproto/googleapis/cloud/vision/v1"
)

type BatchAnnotateResponse map[string]*pb.AnnotateImageResponse

func batchAnnotateURIs(uris []string) (*pb.BatchAnnotateImagesResponse, error) {
	ctx := context.Background()
	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	requests := make([]*pb.AnnotateImageRequest, 0, len(uris))
	for _, uri := range uris {
		requests = append(requests, &pb.AnnotateImageRequest{
			Image: vision.NewImageFromURI(uri),
			Features: []*pb.Feature{
				{Type: pb.Feature_SAFE_SEARCH_DETECTION},
			},
		})
	}

	return client.BatchAnnotateImages(ctx, &pb.BatchAnnotateImagesRequest{Requests: requests})
}

// GetImgSSas returns the SafeSearchAnnotations and any associated errors given uris, and an optional application error.
func GetURIAnnotations(uris []string) ([]*pb.AnnotateImageResponse, error) {
	annotations, err := batchAnnotateURIs(uris)
	if err != nil {
		return nil, err
	}
	return annotations.Responses, nil
}
