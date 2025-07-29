package binder_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/binder"
)

func BenchmarkBindToStruct_SmallStruct(b *testing.B) {
	type SmallStruct struct {
		Field1 string `query:"field1"`
		Field2 int    `query:"field2"`
		Field3 bool   `query:"field3"`
		Field4 string `query:"field4"`
		Field5 int64  `query:"field5"`
	}

	queryBinder := binder.Query()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/?field1=value1&field2=42&field3=true&field4=value4&field5=123456", nil)
		var s SmallStruct
		_ = queryBinder(req, &s)
	}
}

func BenchmarkBindToStruct_LargeStruct(b *testing.B) {
	type LargeStruct struct {
		Field1  string `query:"field1"`
		Field2  string `query:"field2"`
		Field3  string `query:"field3"`
		Field4  string `query:"field4"`
		Field5  string `query:"field5"`
		Field6  string `query:"field6"`
		Field7  string `query:"field7"`
		Field8  string `query:"field8"`
		Field9  string `query:"field9"`
		Field10 string `query:"field10"`
		Field11 string `query:"field11"`
		Field12 string `query:"field12"`
		Field13 string `query:"field13"`
		Field14 string `query:"field14"`
		Field15 string `query:"field15"`
		Field16 string `query:"field16"`
		Field17 string `query:"field17"`
		Field18 string `query:"field18"`
		Field19 string `query:"field19"`
		Field20 string `query:"field20"`
		Field21 string `query:"field21"`
		Field22 string `query:"field22"`
		Field23 string `query:"field23"`
		Field24 string `query:"field24"`
		Field25 string `query:"field25"`
		Field26 string `query:"field26"`
		Field27 string `query:"field27"`
		Field28 string `query:"field28"`
		Field29 string `query:"field29"`
		Field30 string `query:"field30"`
		Field31 string `query:"field31"`
		Field32 string `query:"field32"`
		Field33 string `query:"field33"`
		Field34 string `query:"field34"`
		Field35 string `query:"field35"`
		Field36 string `query:"field36"`
		Field37 string `query:"field37"`
		Field38 string `query:"field38"`
		Field39 string `query:"field39"`
		Field40 string `query:"field40"`
		Field41 string `query:"field41"`
		Field42 string `query:"field42"`
		Field43 string `query:"field43"`
		Field44 string `query:"field44"`
		Field45 string `query:"field45"`
		Field46 string `query:"field46"`
		Field47 string `query:"field47"`
		Field48 string `query:"field48"`
		Field49 string `query:"field49"`
		Field50 string `query:"field50"`
	}

	// Build query string
	queryString := ""
	for i := 1; i <= 50; i++ {
		if i > 1 {
			queryString += "&"
		}
		queryString += "field" + string(rune('0'+i/10)) + string(rune('0'+i%10)) + "=value" + string(rune('0'+i/10)) + string(rune('0'+i%10))
	}

	queryBinder := binder.Query()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/?"+queryString, nil)
		var s LargeStruct
		_ = queryBinder(req, &s)
	}
}

func BenchmarkBindToStruct_MixedTypes(b *testing.B) {
	type MixedStruct struct {
		String   string   `form:"string"`
		Int      int      `form:"int"`
		Int8     int8     `form:"int8"`
		Int16    int16    `form:"int16"`
		Int32    int32    `form:"int32"`
		Int64    int64    `form:"int64"`
		Uint     uint     `form:"uint"`
		Uint8    uint8    `form:"uint8"`
		Uint16   uint16   `form:"uint16"`
		Uint32   uint32   `form:"uint32"`
		Uint64   uint64   `form:"uint64"`
		Float32  float32  `form:"float32"`
		Float64  float64  `form:"float64"`
		Bool     bool     `form:"bool"`
		Slice    []string `form:"slice"`
		IntSlice []int    `form:"intslice"`
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("string", "test")
	writer.WriteField("int", "42")
	writer.WriteField("int8", "127")
	writer.WriteField("int16", "32767")
	writer.WriteField("int32", "2147483647")
	writer.WriteField("int64", "9223372036854775807")
	writer.WriteField("uint", "42")
	writer.WriteField("uint8", "255")
	writer.WriteField("uint16", "65535")
	writer.WriteField("uint32", "4294967295")
	writer.WriteField("uint64", "18446744073709551615")
	writer.WriteField("float32", "3.14")
	writer.WriteField("float64", "3.141592653589793")
	writer.WriteField("bool", "true")
	writer.WriteField("slice", "a")
	writer.WriteField("slice", "b")
	writer.WriteField("slice", "c")
	writer.WriteField("intslice", "1")
	writer.WriteField("intslice", "2")
	writer.WriteField("intslice", "3")
	writer.Close()

	formBinder := binder.Form()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", writer.FormDataContentType())

		var s MixedStruct
		_ = formBinder(req, &s)
	}
}

func BenchmarkBindToStruct_NestedStruct(b *testing.B) {
	type Address struct {
		Street  string `form:"street"`
		City    string `form:"city"`
		State   string `form:"state"`
		Zip     string `form:"zip"`
		Country string `form:"country"`
	}

	type User struct {
		Name      string  `form:"name"`
		Email     string  `form:"email"`
		Age       int     `form:"age"`
		Address   Address // Nested struct
		Phone     string  `form:"phone"`
		Active    bool    `form:"active"`
		CreatedAt string  `form:"created_at"`
		UpdatedAt string  `form:"updated_at"`
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("name", "John Doe")
	writer.WriteField("email", "john@example.com")
	writer.WriteField("age", "30")
	writer.WriteField("phone", "+1234567890")
	writer.WriteField("active", "true")
	writer.WriteField("created_at", "2023-01-01")
	writer.WriteField("updated_at", "2023-01-02")
	writer.Close()

	formBinder := binder.Form()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", writer.FormDataContentType())

		var u User
		_ = formBinder(req, &u)
	}
}

func BenchmarkBindToStruct_PointerFields(b *testing.B) {
	type PointerStruct struct {
		String   *string   `form:"string"`
		Int      *int      `form:"int"`
		Bool     *bool     `form:"bool"`
		Float    *float64  `form:"float"`
		Slice    *[]string `form:"slice"`
		Optional *string   `form:"optional"` // This field won't be set
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("string", "test")
	writer.WriteField("int", "42")
	writer.WriteField("bool", "true")
	writer.WriteField("float", "3.14")
	writer.WriteField("slice", "a")
	writer.WriteField("slice", "b")
	writer.Close()

	formBinder := binder.Form()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", writer.FormDataContentType())

		var s PointerStruct
		_ = formBinder(req, &s)
	}
}

func BenchmarkForm_TagParsing(b *testing.B) {
	// Benchmark the overhead of parsing struct tags
	type TaggedStruct struct {
		DefaultName       string `form:"default_name"`
		CustomName        string `form:"custom"`
		IgnoredField      string `form:"-"`
		EmptyTag          string `form:""`
		ComplexTag        string `form:"complex,omitempty" json:"complex" xml:"complex"`
		AnotherComplexTag string `form:"another,omitempty,required" json:"another" validate:"required"`
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("default_name", "test1")
	writer.WriteField("custom", "test2")
	writer.WriteField("EmptyTag", "test3")
	writer.WriteField("complex", "test4")
	writer.WriteField("another", "test5")
	writer.Close()

	formBinder := binder.Form()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", writer.FormDataContentType())

		var s TaggedStruct
		_ = formBinder(req, &s)
	}
}
