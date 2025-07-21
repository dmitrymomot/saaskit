package binder_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dmitrymomot/saaskit/binder"
)

// createBenchmarkMultipartForm creates a multipart form for benchmarking
func createBenchmarkMultipartForm(files map[string][]byte) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for fieldName, content := range files {
		part, _ := writer.CreateFormFile(fieldName, fieldName+".jpg")
		part.Write(content)
	}

	writer.Close()
	return body, writer.FormDataContentType()
}

func BenchmarkFile_SingleFile(b *testing.B) {
	// Create test data
	fileContent := make([]byte, 1024) // 1KB file
	for i := range fileContent {
		fileContent[i] = byte(i % 256)
	}

	body, contentType := createBenchmarkMultipartForm(map[string][]byte{
		"avatar": fileContent,
	})

	type UploadRequest struct {
		Avatar binder.FileUpload `file:"avatar"`
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create a new reader for each iteration
		req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", contentType)

		var upload UploadRequest
		fileBinder := binder.File()
		_ = fileBinder(req, &upload)
	}
}

func BenchmarkFile_MultipleFiles(b *testing.B) {
	// Create test data with 10 files
	files := make(map[string][]byte)
	fileContent := make([]byte, 1024) // 1KB per file
	for i := 0; i < 10; i++ {
		files[string(rune('a'+i))] = fileContent
	}

	body, contentType := createBenchmarkMultipartForm(files)

	type UploadRequest struct {
		A binder.FileUpload `file:"a"`
		B binder.FileUpload `file:"b"`
		C binder.FileUpload `file:"c"`
		D binder.FileUpload `file:"d"`
		E binder.FileUpload `file:"e"`
		F binder.FileUpload `file:"f"`
		G binder.FileUpload `file:"g"`
		H binder.FileUpload `file:"h"`
		I binder.FileUpload `file:"i"`
		J binder.FileUpload `file:"j"`
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", contentType)

		var upload UploadRequest
		fileBinder := binder.File()
		_ = fileBinder(req, &upload)
	}
}

func BenchmarkFile_LargeStruct(b *testing.B) {
	// Create a struct with many fields to test reflection overhead
	fileContent := make([]byte, 1024)
	body, contentType := createBenchmarkMultipartForm(map[string][]byte{
		"file1": fileContent,
	})

	type LargeUploadRequest struct {
		File1   binder.FileUpload  `file:"file1"`
		File2   *binder.FileUpload `file:"file2"`
		File3   *binder.FileUpload `file:"file3"`
		File4   *binder.FileUpload `file:"file4"`
		File5   *binder.FileUpload `file:"file5"`
		Field6  string             `form:"field6"`
		Field7  string             `form:"field7"`
		Field8  string             `form:"field8"`
		Field9  string             `form:"field9"`
		Field10 string             `form:"field10"`
		Field11 string             `form:"field11"`
		Field12 string             `form:"field12"`
		Field13 string             `form:"field13"`
		Field14 string             `form:"field14"`
		Field15 string             `form:"field15"`
		Field16 string             `form:"field16"`
		Field17 string             `form:"field17"`
		Field18 string             `form:"field18"`
		Field19 string             `form:"field19"`
		Field20 string             `form:"field20"`
		Field21 string             `form:"field21"`
		Field22 string             `form:"field22"`
		Field23 string             `form:"field23"`
		Field24 string             `form:"field24"`
		Field25 string             `form:"field25"`
		Field26 string             `form:"field26"`
		Field27 string             `form:"field27"`
		Field28 string             `form:"field28"`
		Field29 string             `form:"field29"`
		Field30 string             `form:"field30"`
		Field31 string             `form:"field31"`
		Field32 string             `form:"field32"`
		Field33 string             `form:"field33"`
		Field34 string             `form:"field34"`
		Field35 string             `form:"field35"`
		Field36 string             `form:"field36"`
		Field37 string             `form:"field37"`
		Field38 string             `form:"field38"`
		Field39 string             `form:"field39"`
		Field40 string             `form:"field40"`
		Field41 string             `form:"field41"`
		Field42 string             `form:"field42"`
		Field43 string             `form:"field43"`
		Field44 string             `form:"field44"`
		Field45 string             `form:"field45"`
		Field46 string             `form:"field46"`
		Field47 string             `form:"field47"`
		Field48 string             `form:"field48"`
		Field49 string             `form:"field49"`
		Field50 string             `form:"field50"`
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", contentType)

		var upload LargeUploadRequest
		fileBinder := binder.File()
		_ = fileBinder(req, &upload)
	}
}

func BenchmarkFile_ReflectionOverhead(b *testing.B) {
	// Benchmark just the reflection operations without file I/O
	body, contentType := createBenchmarkMultipartForm(map[string][]byte{})

	type EmptyStruct struct {
		Field1  string `file:"field1"`
		Field2  string `file:"field2"`
		Field3  string `file:"field3"`
		Field4  string `file:"field4"`
		Field5  string `file:"field5"`
		Field6  string `file:"field6"`
		Field7  string `file:"field7"`
		Field8  string `file:"field8"`
		Field9  string `file:"field9"`
		Field10 string `file:"field10"`
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", contentType)

		var empty EmptyStruct
		fileBinder := binder.File()
		_ = fileBinder(req, &empty)
	}
}

func BenchmarkGetFile(b *testing.B) {
	fileContent := make([]byte, 1024) // 1KB file
	body, contentType := createBenchmarkMultipartForm(map[string][]byte{
		"avatar": fileContent,
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", contentType)

		_, _ = binder.GetFile(req, "avatar")
	}
}

func BenchmarkGetFiles(b *testing.B) {
	// Create multiple files with same field name
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fileContent := make([]byte, 1024) // 1KB per file
	for i := 0; i < 5; i++ {
		part, _ := writer.CreateFormFile("photos", "photo.jpg")
		part.Write(fileContent)
	}
	writer.Close()
	contentType := writer.FormDataContentType()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", contentType)

		_, _ = binder.GetFiles(req, "photos")
	}
}

func BenchmarkFile_DifferentSizes(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
	}

	type UploadRequest struct {
		File binder.FileUpload `file:"file"`
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			fileContent := make([]byte, tc.size)
			body, contentType := createBenchmarkMultipartForm(map[string][]byte{
				"file": fileContent,
			})

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body.Bytes()))
				req.Header.Set("Content-Type", contentType)

				var upload UploadRequest
				fileBinder := binder.File()
				_ = fileBinder(req, &upload)
			}
		})
	}
}
