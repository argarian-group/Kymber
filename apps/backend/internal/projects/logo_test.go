package projects

import "testing"

func TestDetectLogoType(t *testing.T) {
	cases := map[string]struct {
		data string
		want string
	}{
		"png":         {"\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR", "image/png"},
		"jpeg":        {"\xff\xd8\xff\xe0\x00\x10JFIF", "image/jpeg"},
		"gif":         {"GIF89a\x01\x00\x01\x00", "image/gif"},
		"webp":        {"RIFF\x24\x00\x00\x00WEBPVP8 ", "image/webp"},
		"avif":        {"\x00\x00\x00\x20ftypavif\x00\x00\x00\x00avifmif1", "image/avif"},
		"avif brand":  {"\x00\x00\x00\x20ftypmif1\x00\x00\x00\x00mif1avif", "image/avif"},
		"bmp":         {"BM\x46\x00\x00\x00\x00\x00\x00\x00\x36\x00\x00\x00", "image/bmp"},
		"ico":         {"\x00\x00\x01\x00\x01\x00\x10\x10", "image/x-icon"},
		"svg":         {`<svg xmlns="http://www.w3.org/2000/svg"></svg>`, "image/svg+xml"},
		"svg prolog":  {"\xef\xbb\xbf<?xml version=\"1.0\"?>\n<!-- logo -->\n<svg></svg>", "image/svg+xml"},
		"mp4":         {"\x00\x00\x00\x20ftypisom\x00\x00\x00\x00isommp42", ""},
		"html":        {"<!doctype html><html><script>alert(1)</script></html>", ""},
		"xml not svg": {`<?xml version="1.0"?><note></note>`, ""},
		"text":        {"just some text", ""},
		"pdf":         {"%PDF-1.7\n", ""},
		"empty":       {"", ""},
		"cursor":      {"\x00\x00\x02\x00\x01\x00", ""},
	}
	for name, test := range cases {
		if got := detectLogoType([]byte(test.data)); got != test.want {
			t.Errorf("%s: detectLogoType = %q, want %q", name, got, test.want)
		}
	}
}
