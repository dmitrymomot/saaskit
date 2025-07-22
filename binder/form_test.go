package binder_test

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/binder"
)

func TestForm(t *testing.T) {
	type basicForm struct {
		Name     string  `form:"name"`
		Age      int     `form:"age"`
		Height   float64 `form:"height"`
		Active   bool    `form:"active"`
		Page     uint    `form:"page"`
		Internal string  `form:"-"` // Should be skipped
	}

	t.Run("valid form binding with all types", func(t *testing.T) {
		formData := url.Values{
			"name":   {"John"},
			"age":    {"30"},
			"height": {"5.9"},
			"active": {"true"},
			"page":   {"2"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result basicForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "John", result.Name)
		assert.Equal(t, 30, result.Age)
		assert.Equal(t, 5.9, result.Height)
		assert.Equal(t, true, result.Active)
		assert.Equal(t, uint(2), result.Page)
		assert.Equal(t, "", result.Internal) // Should remain empty
	})

	t.Run("skips fields with dash tag", func(t *testing.T) {
		formData := url.Values{
			"name":     {"Test"},
			"internal": {"secret"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result basicForm
		result.Internal = "original" // Set a value that should not be overwritten
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "Test", result.Name)
		assert.Equal(t, "original", result.Internal) // Should not be changed
	})

	t.Run("content type with charset", func(t *testing.T) {
		formData := url.Values{
			"name": {"Jane"},
			"age":  {"25"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

		var result basicForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "Jane", result.Name)
		assert.Equal(t, 25, result.Age)
	})

	t.Run("missing content type", func(t *testing.T) {
		formData := url.Values{"name": {"Test"}}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		// Don't set Content-Type

		var result basicForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.True(t, errors.Is(err, binder.ErrMissingContentType))
		assert.Contains(t, err.Error(), "expected application/x-www-form-urlencoded")
	})

	t.Run("wrong content type", func(t *testing.T) {
		formData := url.Values{"name": {"Test"}}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/json")

		var result basicForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.True(t, errors.Is(err, binder.ErrUnsupportedMediaType))
		assert.Contains(t, err.Error(), "got application/json")
	})

	t.Run("empty form data", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result basicForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "", result.Name)      // zero value
		assert.Equal(t, 0, result.Age)        // zero value
		assert.Equal(t, 0.0, result.Height)   // zero value
		assert.Equal(t, false, result.Active) // zero value
		assert.Equal(t, uint(0), result.Page) // zero value
	})

	t.Run("partial form data", func(t *testing.T) {
		formData := url.Values{
			"name": {"Jane"},
			"age":  {"25"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result basicForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "Jane", result.Name)
		assert.Equal(t, 25, result.Age)
		assert.Equal(t, 0.0, result.Height)   // zero value
		assert.Equal(t, false, result.Active) // zero value
	})

	t.Run("invalid int value", func(t *testing.T) {
		formData := url.Values{"age": {"notanumber"}}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result basicForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid int value")
		assert.Contains(t, err.Error(), "Age")
	})

	t.Run("slice parameters multiple values", func(t *testing.T) {
		type sliceForm struct {
			Tags []string `form:"tags"`
			IDs  []int    `form:"ids"`
		}

		formData := url.Values{
			"tags": {"go", "web", "api"},
			"ids":  {"1", "2", "3"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result sliceForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, []string{"go", "web", "api"}, result.Tags)
		assert.Equal(t, []int{1, 2, 3}, result.IDs)
	})

	t.Run("slice parameters comma separated", func(t *testing.T) {
		type sliceForm struct {
			Tags   []string  `form:"tags"`
			Scores []float64 `form:"scores"`
		}

		formData := url.Values{
			"tags":   {"go,web,api"},
			"scores": {"1.5,2.0,3.5"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result sliceForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, []string{"go", "web", "api"}, result.Tags)
		assert.Equal(t, []float64{1.5, 2.0, 3.5}, result.Scores)
	})

	t.Run("boolean slice parameters", func(t *testing.T) {
		type boolSliceForm struct {
			Flags []bool `form:"flags"`
		}

		formData := url.Values{
			"flags": {"true", "false", "1", "0", "yes", "no", "on", "off"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result boolSliceForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, []bool{true, false, true, false, true, false, true, false}, result.Flags)
	})

	t.Run("boolean slice comma separated", func(t *testing.T) {
		type boolSliceForm struct {
			Settings []bool `form:"settings"`
		}

		formData := url.Values{
			"settings": {"true,false,yes,no,1,0,ON,OFF"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result boolSliceForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, []bool{true, false, true, false, true, false, true, false}, result.Settings)
	})

	t.Run("pointer fields", func(t *testing.T) {
		type pointerForm struct {
			Name     *string  `form:"name"`
			Age      *int     `form:"age"`
			Active   *bool    `form:"active"`
			Score    *float64 `form:"score"`
			Required string   `form:"required"`
		}

		formData := url.Values{
			"name":     {"John"},
			"active":   {"true"},
			"required": {"value"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result pointerForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		require.NotNil(t, result.Name)
		assert.Equal(t, "John", *result.Name)
		assert.Nil(t, result.Age) // Not provided
		require.NotNil(t, result.Active)
		assert.Equal(t, true, *result.Active)
		assert.Nil(t, result.Score) // Not provided
		assert.Equal(t, "value", result.Required)
	})

	t.Run("no struct tag uses lowercase field name", func(t *testing.T) {
		type noTagForm struct {
			Name  string
			Count int
		}

		formData := url.Values{
			"name":  {"Test"},
			"count": {"5"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result noTagForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "Test", result.Name)
		assert.Equal(t, 5, result.Count)
	})

	t.Run("special characters in values", func(t *testing.T) {
		type specialForm struct {
			Email   string `form:"email"`
			URL     string `form:"url"`
			Message string `form:"msg"`
		}

		formData := url.Values{
			"email": {"user@example.com"},
			"url":   {"https://example.com?q=test&p=1"},
			"msg":   {"Hello World!"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result specialForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "user@example.com", result.Email)
		assert.Equal(t, "https://example.com?q=test&p=1", result.URL)
		assert.Equal(t, "Hello World!", result.Message)
	})

	t.Run("non-pointer target", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result basicForm
		bindFunc := binder.Form()
		err := bindFunc(req, result) // Pass by value, not pointer

		require.Error(t, err)
		assert.Contains(t, err.Error(), "target must be a non-nil pointer")
	})

	t.Run("boolean variations", func(t *testing.T) {
		tests := []struct {
			value    string
			expected bool
		}{
			// Standard boolean strings
			{"true", true},
			{"false", false},
			{"True", true},
			{"False", false},
			{"TRUE", true},
			{"FALSE", false},
			{"t", true},
			{"f", false},
			{"T", true},
			{"F", false},

			// Numeric strings
			{"1", true},
			{"0", false},

			// Alternative boolean strings
			{"on", true},
			{"off", false},
			{"On", true},
			{"Off", false},
			{"ON", true},
			{"OFF", false},

			{"yes", true},
			{"no", false},
			{"Yes", true},
			{"No", false},
			{"YES", true},
			{"NO", false},

			// Empty value (tested separately in "empty values in form" test)
		}

		for _, tt := range tests {
			t.Run(tt.value, func(t *testing.T) {
				formData := url.Values{"active": {tt.value}}
				req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

				var result basicForm
				bindFunc := binder.Form()
				err := bindFunc(req, &result)

				require.NoError(t, err)
				assert.Equal(t, tt.expected, result.Active)
			})
		}
	})

	t.Run("invalid boolean values", func(t *testing.T) {
		invalidValues := []string{
			"maybe",
			"unknown",
			"y",
			"n",
			"Y",
			"N",
			"2",
			"-1",
			"10",
			"truee",
			"fals",
			"yess",
			"noo",
		}

		for _, value := range invalidValues {
			t.Run(value, func(t *testing.T) {
				formData := url.Values{"active": {value}}
				req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

				var result basicForm
				bindFunc := binder.Form()
				err := bindFunc(req, &result)

				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid bool value")
				assert.Contains(t, err.Error(), value)
			})
		}
	})

	t.Run("tags with options", func(t *testing.T) {
		type tagOptionsForm struct {
			Name     string `form:"name,omitempty"`
			Optional string `form:"opt,omitempty"`
			Count    int    `form:"count,omitempty"`
		}

		formData := url.Values{
			"name":  {"Test"},
			"count": {"10"},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result tagOptionsForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "Test", result.Name)
		assert.Equal(t, "", result.Optional) // Not provided, zero value
		assert.Equal(t, 10, result.Count)
	})

	t.Run("enterprise application settings form", func(t *testing.T) {
		// Realistic large configuration form for enterprise SaaS application
		type AppSettings struct {
			// General Settings
			AppName         string `form:"app_name"`
			AppURL          string `form:"app_url"`
			SupportEmail    string `form:"support_email"`
			DefaultTimezone string `form:"default_timezone"`
			DefaultLanguage string `form:"default_language"`
			MaintenanceMode bool   `form:"maintenance_mode"`

			// Security Settings
			RequireSSL          bool     `form:"require_ssl"`
			SessionTimeout      int      `form:"session_timeout"`
			MaxLoginAttempts    int      `form:"max_login_attempts"`
			PasswordMinLength   int      `form:"password_min_length"`
			RequireUppercase    bool     `form:"require_uppercase"`
			RequireLowercase    bool     `form:"require_lowercase"`
			RequireNumbers      bool     `form:"require_numbers"`
			RequireSpecialChars bool     `form:"require_special_chars"`
			TwoFactorAuth       string   `form:"two_factor_auth"`
			AllowedIPRanges     []string `form:"allowed_ip_ranges"`

			// Email Settings
			SMTPHost         string `form:"smtp_host"`
			SMTPPort         int    `form:"smtp_port"`
			SMTPUser         string `form:"smtp_user"`
			SMTPPassword     string `form:"smtp_password"`
			SMTPEncryption   string `form:"smtp_encryption"`
			EmailFromName    string `form:"email_from_name"`
			EmailFromAddress string `form:"email_from_address"`

			// Storage Settings
			StorageProvider  string   `form:"storage_provider"`
			S3Bucket         string   `form:"s3_bucket"`
			S3Region         string   `form:"s3_region"`
			S3AccessKey      string   `form:"s3_access_key"`
			S3SecretKey      string   `form:"s3_secret_key"`
			MaxUploadSize    int      `form:"max_upload_size"`
			AllowedFileTypes []string `form:"allowed_file_types"`

			// API Settings
			APIRateLimit   int      `form:"api_rate_limit"`
			APIBurstLimit  int      `form:"api_burst_limit"`
			WebhookTimeout int      `form:"webhook_timeout"`
			APIVersions    []string `form:"api_versions"`

			// Feature Flags
			EnableSignup      bool `form:"enable_signup"`
			EnableGuestAccess bool `form:"enable_guest_access"`
			EnableAPIAccess   bool `form:"enable_api_access"`
			EnableWebhooks    bool `form:"enable_webhooks"`
			EnableExports     bool `form:"enable_exports"`
			EnableImports     bool `form:"enable_imports"`

			// Logging Settings
			LogLevel         string   `form:"log_level"`
			LogRetentionDays int      `form:"log_retention_days"`
			EnableAuditLog   bool     `form:"enable_audit_log"`
			AuditLogEvents   []string `form:"audit_log_events"`
		}

		// Create realistic form data
		formData := url.Values{
			// General
			"app_name":         {"Enterprise SaaS Platform"},
			"app_url":          {"https://app.enterprise.com"},
			"support_email":    {"support@enterprise.com"},
			"default_timezone": {"America/New_York"},
			"default_language": {"en-US"},
			"maintenance_mode": {"false"},

			// Security
			"require_ssl":           {"true"},
			"session_timeout":       {"3600"},
			"max_login_attempts":    {"5"},
			"password_min_length":   {"12"},
			"require_uppercase":     {"true"},
			"require_lowercase":     {"true"},
			"require_numbers":       {"true"},
			"require_special_chars": {"true"},
			"two_factor_auth":       {"required"},
			"allowed_ip_ranges":     {"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},

			// Email
			"smtp_host":          {"smtp.sendgrid.net"},
			"smtp_port":          {"587"},
			"smtp_user":          {"apikey"},
			"smtp_password":      {"SG.xxxxxxxxxxxx"},
			"smtp_encryption":    {"tls"},
			"email_from_name":    {"Enterprise Platform"},
			"email_from_address": {"noreply@enterprise.com"},

			// Storage
			"storage_provider":   {"s3"},
			"s3_bucket":          {"enterprise-uploads"},
			"s3_region":          {"us-east-1"},
			"s3_access_key":      {"AKIAXXXXXXXXXXXXXXXX"},
			"s3_secret_key":      {"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"},
			"max_upload_size":    {"104857600"}, // 100MB
			"allowed_file_types": {"pdf", "doc", "docx", "xls", "xlsx", "png", "jpg", "jpeg"},

			// API
			"api_rate_limit":  {"1000"},
			"api_burst_limit": {"2000"},
			"webhook_timeout": {"30"},
			"api_versions":    {"v1", "v2", "v3"},

			// Features
			"enable_signup":       {"true"},
			"enable_guest_access": {"false"},
			"enable_api_access":   {"true"},
			"enable_webhooks":     {"true"},
			"enable_exports":      {"true"},
			"enable_imports":      {"true"},

			// Logging
			"log_level":          {"info"},
			"log_retention_days": {"90"},
			"enable_audit_log":   {"true"},
			"audit_log_events":   {"login", "logout", "create", "update", "delete", "export"},
		}

		req := httptest.NewRequest(http.MethodPost, "/admin/settings", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result AppSettings
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		// Verify a sample of fields to ensure proper binding
		assert.Equal(t, "Enterprise SaaS Platform", result.AppName)
		assert.True(t, result.RequireSSL)
		assert.Equal(t, 3600, result.SessionTimeout)
		assert.Equal(t, []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}, result.AllowedIPRanges)
		assert.Equal(t, "s3", result.StorageProvider)
		assert.Equal(t, 104857600, result.MaxUploadSize)
		assert.Equal(t, []string{"pdf", "doc", "docx", "xls", "xlsx", "png", "jpg", "jpeg"}, result.AllowedFileTypes)
		assert.True(t, result.EnableAuditLog)
		assert.Equal(t, []string{"login", "logout", "create", "update", "delete", "export"}, result.AuditLogEvents)
	})

	t.Run("invalid form data", func(t *testing.T) {
		// Send malformed form data
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString("%ZZ"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result basicForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.True(t, errors.Is(err, binder.ErrInvalidForm))
	})

	t.Run("empty values in form", func(t *testing.T) {
		formData := url.Values{
			"name":   {""},
			"active": {""},
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result basicForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "", result.Name)
		assert.Equal(t, false, result.Active) // Empty string is treated as false
	})

	t.Run("multipart form content type", func(t *testing.T) {
		// Create a proper multipart form
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		_ = w.WriteField("name", "Test")
		_ = w.WriteField("age", "25")
		_ = w.Close()

		req := httptest.NewRequest(http.MethodPost, "/test", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())

		var result basicForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		// Form now supports multipart forms
		require.NoError(t, err)
		assert.Equal(t, "Test", result.Name)
		assert.Equal(t, 25, result.Age)
	})
}

func TestFormWithFiles(t *testing.T) {
	type uploadForm struct {
		Title    string                  `form:"title"`
		Category string                  `form:"category"`
		Avatar   *multipart.FileHeader   `file:"avatar"`
		Gallery  []*multipart.FileHeader `file:"gallery"`
		Document *multipart.FileHeader   `file:"document"`
		Skip     *multipart.FileHeader   `file:"-"`
		NoTag    *multipart.FileHeader
		private  *multipart.FileHeader `file:"private"`
	}

	t.Run("form and file fields together", func(t *testing.T) {
		body, contentType := createMultipartFormWithFiles(t,
			map[string]string{
				"title":    "My Upload",
				"category": "photos",
			},
			map[string][]fileData{
				"avatar": {{filename: "avatar.jpg", content: []byte("avatar data")}},
			},
		)

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		var result uploadForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "My Upload", result.Title)
		assert.Equal(t, "photos", result.Category)
		require.NotNil(t, result.Avatar)
		assert.Equal(t, "avatar.jpg", result.Avatar.Filename)
		assert.Equal(t, int64(11), result.Avatar.Size) // "avatar data" is 11 bytes
	})

	t.Run("optional file present", func(t *testing.T) {
		body, contentType := createMultipartFormWithFiles(t,
			map[string]string{"title": "Test"},
			map[string][]fileData{
				"document": {{filename: "doc.pdf", content: []byte("pdf content")}},
			},
		)

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		var result uploadForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		require.NotNil(t, result.Document)
		assert.Equal(t, "doc.pdf", result.Document.Filename)
		assert.Nil(t, result.Avatar) // Not provided
	})

	t.Run("multiple files", func(t *testing.T) {
		body, contentType := createMultipartFormWithFiles(t,
			map[string]string{"title": "Gallery"},
			map[string][]fileData{
				"gallery": {
					{filename: "img1.jpg", content: []byte("image1")},
					{filename: "img2.jpg", content: []byte("image2")},
					{filename: "img3.jpg", content: []byte("image3")},
				},
			},
		)

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		var result uploadForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		require.Len(t, result.Gallery, 3)
		assert.Equal(t, "img1.jpg", result.Gallery[0].Filename)
		assert.Equal(t, "img2.jpg", result.Gallery[1].Filename)
		assert.Equal(t, "img3.jpg", result.Gallery[2].Filename)
	})

	t.Run("skip fields with dash tag", func(t *testing.T) {
		body, contentType := createMultipartFormWithFiles(t,
			map[string]string{"title": "Test"},
			map[string][]fileData{
				"skip": {{filename: "skip.txt", content: []byte("should not bind")}},
			},
		)

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		var result uploadForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Nil(t, result.Skip)
	})

	t.Run("filename sanitization", func(t *testing.T) {
		dangerousFilenames := []struct {
			input    string
			expected string
		}{
			{"../../../etc/passwd", "passwd"},
			{"..\\..\\windows\\system32\\config", "config"},
			{"/etc/passwd", "passwd"},
			{"C:\\Windows\\System32\\config.sys", "config.sys"},
			{".", "unnamed"},
			{"..", "unnamed"},
			{"/", "unnamed"},
			{"normal.txt", "normal.txt"},
		}

		for _, tc := range dangerousFilenames {
			t.Run(tc.input, func(t *testing.T) {
				body, contentType := createMultipartFormWithFiles(t,
					map[string]string{"title": "Test"},
					map[string][]fileData{
						"avatar": {{filename: tc.input, content: []byte("data")}},
					},
				)

				req := httptest.NewRequest(http.MethodPost, "/upload", body)
				req.Header.Set("Content-Type", contentType)

				var result uploadForm
				bindFunc := binder.Form()
				err := bindFunc(req, &result)

				require.NoError(t, err)
				require.NotNil(t, result.Avatar)
				assert.Equal(t, tc.expected, result.Avatar.Filename)
			})
		}
	})

	t.Run("unsupported file field type", func(t *testing.T) {
		type invalidForm struct {
			File string `file:"file"` // Wrong type
		}

		body, contentType := createMultipartFormWithFiles(t,
			nil,
			map[string][]fileData{
				"file": {{filename: "test.txt", content: []byte("data")}},
			},
		)

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", contentType)

		var result invalidForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported type for file field")
	})

	t.Run("url-encoded form skips file tags", func(t *testing.T) {
		// File tags should be ignored for non-multipart forms
		formData := url.Values{
			"title":  {"Test"},
			"avatar": {"ignored"}, // This should be ignored
		}
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result uploadForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "Test", result.Title)
		assert.Nil(t, result.Avatar) // File field should remain nil
	})
}

// Helper types and functions for file tests

type fileData struct {
	filename string
	content  []byte
}

func createMultipartFormWithFiles(t *testing.T, fields map[string]string, files map[string][]fileData) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	for name, value := range fields {
		err := writer.WriteField(name, value)
		require.NoError(t, err)
	}

	// Add files
	for fieldName, fieldFiles := range files {
		for _, file := range fieldFiles {
			part, err := writer.CreateFormFile(fieldName, file.filename)
			require.NoError(t, err)
			_, err = part.Write(file.content)
			require.NoError(t, err)
		}
	}

	err := writer.Close()
	require.NoError(t, err)

	return body, writer.FormDataContentType()
}
