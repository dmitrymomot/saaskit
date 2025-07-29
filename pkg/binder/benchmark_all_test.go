package binder_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/binder"
)

func BenchmarkCombinedBinders(b *testing.B) {
	// Simulate a complex request with JSON body, query params, and files
	type ComplexRequest struct {
		// From query string
		Page     int    `query:"page"`
		PageSize int    `query:"page_size"`
		Sort     string `query:"sort"`

		// From path params
		UserID string `path:"user_id"`

		// From JSON body
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`

		// From form data
		Category string `form:"category"`
		Priority int    `form:"priority"`

		// File uploads
		Avatar  *multipart.FileHeader   `file:"avatar"`
		Gallery []*multipart.FileHeader `file:"gallery"`
	}

	// Create multipart body with JSON and files
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	writer.WriteField("category", "business")
	writer.WriteField("priority", "1")

	// Add JSON part
	jsonData := map[string]any{
		"name":        "Test Product",
		"description": "A test product description",
		"tags":        []string{"tag1", "tag2", "tag3"},
	}
	jsonBytes, _ := json.Marshal(jsonData)
	jsonPart, _ := writer.CreateFormField("json")
	jsonPart.Write(jsonBytes)

	// Add files
	avatarPart, _ := writer.CreateFormFile("avatar", "avatar.jpg")
	avatarPart.Write(make([]byte, 1024)) // 1KB avatar

	for range 3 {
		galleryPart, _ := writer.CreateFormFile("gallery", "photo.jpg")
		galleryPart.Write(make([]byte, 1024)) // 1KB per gallery image
	}

	writer.Close()

	// Create binders
	queryBinder := binder.Query()
	pathBinder := binder.Path(func(r *http.Request, fieldName string) string {
		// Simulate path param extraction
		if fieldName == "user_id" {
			return "123"
		}
		return ""
	})
	formBinder := binder.Form()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		req := httptest.NewRequest(
			http.MethodPost,
			"/users/123/products?page=1&page_size=20&sort=name",
			bytes.NewReader(body.Bytes()),
		)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		// Path params are handled by the path extractor function

		var complexReq ComplexRequest

		// Apply all binders
		_ = queryBinder(req, &complexReq)
		_ = pathBinder(req, &complexReq)
		_ = formBinder(req, &complexReq)

		// Note: JSON binding would typically be done on a separate request
		// since you can't have both multipart and JSON body at the same time
	}
}

func BenchmarkRealWorldScenario_UserProfile(b *testing.B) {
	// Simulate a real-world user profile update
	type UserProfileUpdate struct {
		// Basic info from form
		FirstName   string `form:"first_name"`
		LastName    string `form:"last_name"`
		Email       string `form:"email"`
		Phone       string `form:"phone"`
		Bio         string `form:"bio"`
		DateOfBirth string `form:"date_of_birth"`

		// Address info from form
		Street     string `form:"street"`
		City       string `form:"city"`
		State      string `form:"state"`
		PostalCode string `form:"postal_code"`
		Country    string `form:"country"`

		// Settings from form
		Newsletter    bool   `form:"newsletter"`
		EmailUpdates  bool   `form:"email_updates"`
		PublicProfile bool   `form:"public_profile"`
		Timezone      string `form:"timezone"`
		Language      string `form:"language"`

		// File uploads
		ProfilePicture *multipart.FileHeader `file:"profile_picture"`
		Resume         *multipart.FileHeader `file:"resume"`
	}

	// Create realistic form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	fields := map[string]string{
		"first_name":     "John",
		"last_name":      "Doe",
		"email":          "john.doe@example.com",
		"phone":          "+1234567890",
		"bio":            "Software engineer with 10 years of experience in building scalable web applications.",
		"date_of_birth":  "1990-01-01",
		"street":         "123 Main St",
		"city":           "San Francisco",
		"state":          "CA",
		"postal_code":    "94105",
		"country":        "USA",
		"newsletter":     "true",
		"email_updates":  "true",
		"public_profile": "false",
		"timezone":       "America/Los_Angeles",
		"language":       "en",
	}

	for k, v := range fields {
		writer.WriteField(k, v)
	}

	// Add profile picture
	picPart, _ := writer.CreateFormFile("profile_picture", "profile.jpg")
	picPart.Write(make([]byte, 100*1024)) // 100KB profile picture

	// Add resume
	resumePart, _ := writer.CreateFormFile("resume", "resume.pdf")
	resumePart.Write(make([]byte, 200*1024)) // 200KB resume

	writer.Close()

	formBinder := binder.Form()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/profile/update", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", writer.FormDataContentType())

		var profile UserProfileUpdate
		_ = formBinder(req, &profile)
	}
}

func BenchmarkRealWorldScenario_ProductListing(b *testing.B) {
	// Simulate a product listing creation
	type ProductListing struct {
		// From JSON body
		Title           string   `json:"title"`
		Description     string   `json:"description"`
		Price           float64  `json:"price"`
		Currency        string   `json:"currency"`
		Stock           int      `json:"stock"`
		Categories      []string `json:"categories"`
		Tags            []string `json:"tags"`
		Features        []string `json:"features"`
		ShippingOptions []string `json:"shipping_options"`

		// From query params
		Draft   bool `query:"draft"`
		Preview bool `query:"preview"`

		// From path params
		StoreID string `path:"store_id"`

		// File uploads
		MainImage      *multipart.FileHeader   `file:"main_image"`
		Images         []*multipart.FileHeader `file:"images"`
		Specifications *multipart.FileHeader   `file:"specifications"`
	}

	// Create request body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add JSON data as form field
	jsonData := map[string]any{
		"title":       "Premium Wireless Headphones",
		"description": "High-quality wireless headphones with noise cancellation and 30-hour battery life.",
		"price":       299.99,
		"currency":    "USD",
		"stock":       100,
		"categories":  []string{"Electronics", "Audio", "Headphones"},
		"tags":        []string{"wireless", "bluetooth", "noise-cancelling", "premium"},
		"features": []string{
			"Active Noise Cancellation",
			"30-hour battery life",
			"Bluetooth 5.0",
			"Comfortable over-ear design",
			"Built-in microphone",
		},
		"shipping_options": []string{"standard", "express", "overnight"},
	}
	jsonBytes, _ := json.Marshal(jsonData)
	jsonPart, _ := writer.CreateFormField("data")
	jsonPart.Write(jsonBytes)

	// Add main image
	mainImgPart, _ := writer.CreateFormFile("main_image", "main.jpg")
	mainImgPart.Write(make([]byte, 500*1024)) // 500KB main image

	// Add gallery images
	for range 5 {
		imgPart, _ := writer.CreateFormFile("images", "image.jpg")
		imgPart.Write(make([]byte, 300*1024)) // 300KB per image
	}

	// Add specifications PDF
	specPart, _ := writer.CreateFormFile("specifications", "specs.pdf")
	specPart.Write(make([]byte, 1024*1024)) // 1MB specifications

	writer.Close()

	queryBinder := binder.Query()
	pathBinder := binder.Path(func(r *http.Request, fieldName string) string {
		if fieldName == "store_id" {
			return "store123"
		}
		return ""
	})
	jsonBinder := binder.JSON()
	formBinder := binder.Form()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		// Create request with JSON body first
		jsonReq := httptest.NewRequest(http.MethodPost, "/stores/store123/products?draft=false&preview=true", bytes.NewReader(jsonBytes))
		jsonReq.Header.Set("Content-Type", "application/json")

		// Create multipart request for files
		fileReq := httptest.NewRequest(http.MethodPost, "/stores/store123/products?draft=false&preview=true", bytes.NewReader(body.Bytes()))
		fileReq.Header.Set("Content-Type", writer.FormDataContentType())

		// Path params are handled by the path extractor function

		var product ProductListing

		// Apply binders
		_ = queryBinder(fileReq, &product)
		_ = pathBinder(fileReq, &product)
		_ = jsonBinder(jsonReq, &product)
		_ = formBinder(fileReq, &product)
	}
}
