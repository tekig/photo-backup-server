package photo

import (
	"os"
	"testing"
)

func Test_makePreview(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		contentType string
	}{
		{
			name:        "video",
			source:      "https://storage.yandexcloud.net/belld-standart/origin/IMG_20250215_092743_664.heic?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=YCAJEq5eMQ_wmzYDrrcg8m7i2%2F20250518%2Fru-central1%2Fs3%2Faws4_request&X-Amz-Date=20250518T095544Z&X-Amz-Expires=3600&X-Amz-Signature=96e510c3b6b8e9dfb6b8d15bff5497534199cf2e7e2073e5c84358416bf5ccd0&X-Amz-SignedHeaders=host",
			contentType: "image/",
		},
		{
			name:        "video",
			source:      "https://docs.evostream.com/sample_content/assets/bunny.mp4",
			contentType: "video/mp4",
		},
		{
			name:        "image",
			source:      "https://picsum.photos/2000/1000",
			contentType: "image/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, ct, err := makePreview(t.Context(), tt.source, &tt.contentType)
			if err != nil {
				t.Errorf("makePreview() error = %v", err)
				return
			}
			if err := os.Remove(p); err != nil {
				t.Errorf("removePreview() error = %v", err)
				return
			}
			_ = ct
		})
	}
}
