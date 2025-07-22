package binder_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/binder"
)

func TestRealWorldSaaSQueryScenarios(t *testing.T) {
	t.Run("pagination parameters", func(t *testing.T) {
		type PaginationParams struct {
			Page     int    `query:"page"`
			Limit    int    `query:"limit"`
			Offset   int    `query:"offset"`
			Cursor   string `query:"cursor"`
			PageSize int    `query:"page_size"`
		}

		tests := []struct {
			name     string
			query    string
			expected PaginationParams
		}{
			{
				name:  "standard page and limit",
				query: "page=2&limit=50",
				expected: PaginationParams{
					Page:  2,
					Limit: 50,
				},
			},
			{
				name:  "offset-based pagination",
				query: "offset=100&limit=25",
				expected: PaginationParams{
					Offset: 100,
					Limit:  25,
				},
			},
			{
				name:  "cursor-based pagination",
				query: "cursor=eyJpZCI6MTIzfQ&page_size=20",
				expected: PaginationParams{
					Cursor:   "eyJpZCI6MTIzfQ",
					PageSize: 20,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/api/users?"+tt.query, nil)

				var result PaginationParams
				bindFunc := binder.Query()
				err := bindFunc(req, &result)

				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("sorting and filtering", func(t *testing.T) {
		type FilterParams struct {
			SortBy   string   `query:"sort_by"`
			Order    string   `query:"order"`
			Status   []string `query:"status"`
			Tags     []string `query:"tags"`
			Search   string   `query:"q"`
			MinPrice float64  `query:"min_price"`
			MaxPrice float64  `query:"max_price"`
			InStock  *bool    `query:"in_stock"`
			Featured bool     `query:"featured"`
		}

		req := httptest.NewRequest(http.MethodGet,
			"/api/products?sort_by=price&order=desc&status=active,pending&tags=electronics,sale&q=laptop&min_price=500&max_price=2000&in_stock=true&featured=false",
			nil)

		var result FilterParams
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "price", result.SortBy)
		assert.Equal(t, "desc", result.Order)
		assert.Equal(t, []string{"active", "pending"}, result.Status)
		assert.Equal(t, []string{"electronics", "sale"}, result.Tags)
		assert.Equal(t, "laptop", result.Search)
		assert.Equal(t, 500.0, result.MinPrice)
		assert.Equal(t, 2000.0, result.MaxPrice)
		require.NotNil(t, result.InStock)
		assert.True(t, *result.InStock)
		assert.False(t, result.Featured)
	})

	t.Run("date range filtering", func(t *testing.T) {
		type DateRangeParams struct {
			StartDate    string `query:"start_date"`
			EndDate      string `query:"end_date"`
			CreatedAfter string `query:"created_after"`
			UpdatedSince string `query:"updated_since"`
			DateFrom     string `query:"date_from"`
			DateTo       string `query:"date_to"`
		}

		req := httptest.NewRequest(http.MethodGet,
			"/api/reports?start_date=2024-01-01&end_date=2024-12-31&created_after=2024-01-01T00:00:00Z&updated_since=2024-06-01",
			nil)

		var result DateRangeParams
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "2024-01-01", result.StartDate)
		assert.Equal(t, "2024-12-31", result.EndDate)
		assert.Equal(t, "2024-01-01T00:00:00Z", result.CreatedAfter)
		assert.Equal(t, "2024-06-01", result.UpdatedSince)
	})

	t.Run("complex search with includes and excludes", func(t *testing.T) {
		type SearchParams struct {
			Query      string   `query:"q"`
			Include    []string `query:"include"`
			Exclude    []string `query:"exclude"`
			Fields     []string `query:"fields"`
			Expand     []string `query:"expand"`
			ExcludeIDs []int    `query:"exclude_ids"`
			Categories []string `query:"category"`
		}

		req := httptest.NewRequest(http.MethodGet,
			"/api/search?q=golang&include=tutorials,examples&exclude=deprecated&fields=title,description,author&expand=author,tags&exclude_ids=123,456,789&category=backend,web",
			nil)

		var result SearchParams
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "golang", result.Query)
		assert.Equal(t, []string{"tutorials", "examples"}, result.Include)
		assert.Equal(t, []string{"deprecated"}, result.Exclude)
		assert.Equal(t, []string{"title", "description", "author"}, result.Fields)
		assert.Equal(t, []string{"author", "tags"}, result.Expand)
		assert.Equal(t, []int{123, 456, 789}, result.ExcludeIDs)
		assert.Equal(t, []string{"backend", "web"}, result.Categories)
	})

	t.Run("analytics and metrics parameters", func(t *testing.T) {
		type AnalyticsParams struct {
			MetricType  string   `query:"metric"`
			GroupBy     []string `query:"group_by"`
			Interval    string   `query:"interval"`
			Timezone    string   `query:"tz"`
			Aggregate   string   `query:"aggregate"`
			Breakdown   []string `query:"breakdown"`
			Percentiles []int    `query:"percentiles"`
		}

		req := httptest.NewRequest(http.MethodGet,
			"/api/analytics?metric=revenue&group_by=country,product&interval=daily&tz=America/New_York&aggregate=sum&breakdown=channel,device&percentiles=50,90,99",
			nil)

		var result AnalyticsParams
		bindFunc := binder.Query()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "revenue", result.MetricType)
		assert.Equal(t, []string{"country", "product"}, result.GroupBy)
		assert.Equal(t, "daily", result.Interval)
		assert.Equal(t, "America/New_York", result.Timezone)
		assert.Equal(t, "sum", result.Aggregate)
		assert.Equal(t, []string{"channel", "device"}, result.Breakdown)
		assert.Equal(t, []int{50, 90, 99}, result.Percentiles)
	})
}

func TestRealWorldSaaSFormScenarios(t *testing.T) {
	t.Run("user profile update form", func(t *testing.T) {
		type UserProfileForm struct {
			FirstName     string   `form:"first_name"`
			LastName      string   `form:"last_name"`
			Email         string   `form:"email"`
			Phone         string   `form:"phone"`
			Bio           string   `form:"bio"`
			Website       string   `form:"website"`
			Company       string   `form:"company"`
			JobTitle      string   `form:"job_title"`
			Location      string   `form:"location"`
			Skills        []string `form:"skills"`
			Interests     []string `form:"interests"`
			NotifyEmail   bool     `form:"notify_email"`
			NotifySMS     bool     `form:"notify_sms"`
			PublicProfile bool     `form:"public_profile"`
			TwoFactorAuth bool     `form:"two_factor_auth"`
			PreferredLang string   `form:"preferred_language"`
		}

		formData := url.Values{
			"first_name":         {"John"},
			"last_name":          {"Doe"},
			"email":              {"john.doe@company.com"},
			"phone":              {"+1-555-123-4567"},
			"bio":                {"Senior software engineer with 10+ years of experience"},
			"website":            {"https://johndoe.dev"},
			"company":            {"TechCorp Inc."},
			"job_title":          {"Senior Software Engineer"},
			"location":           {"San Francisco, CA"},
			"skills":             {"Go", "Python", "Kubernetes", "Docker"},
			"interests":          {"Cloud Architecture", "DevOps", "Machine Learning"},
			"notify_email":       {"true"},
			"notify_sms":         {"false"},
			"public_profile":     {"true"},
			"two_factor_auth":    {"true"},
			"preferred_language": {"en-US"},
		}

		req := httptest.NewRequest(http.MethodPost, "/profile/update", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result UserProfileForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "John", result.FirstName)
		assert.Equal(t, "john.doe@company.com", result.Email)
		assert.Equal(t, []string{"Go", "Python", "Kubernetes", "Docker"}, result.Skills)
		assert.True(t, result.NotifyEmail)
		assert.False(t, result.NotifySMS)
		assert.True(t, result.TwoFactorAuth)
	})

	t.Run("subscription checkout form", func(t *testing.T) {
		type CheckoutForm struct {
			PlanID         string   `form:"plan_id"`
			BillingCycle   string   `form:"billing_cycle"`
			PaymentMethod  string   `form:"payment_method"`
			CardNumber     string   `form:"card_number"`
			CardExpiry     string   `form:"card_expiry"`
			CardCVC        string   `form:"card_cvc"`
			BillingName    string   `form:"billing_name"`
			BillingEmail   string   `form:"billing_email"`
			BillingAddress string   `form:"billing_address"`
			BillingCity    string   `form:"billing_city"`
			BillingState   string   `form:"billing_state"`
			BillingZip     string   `form:"billing_zip"`
			BillingCountry string   `form:"billing_country"`
			CouponCode     string   `form:"coupon_code"`
			AddOns         []string `form:"addons"`
			AutoRenew      bool     `form:"auto_renew"`
			AcceptTerms    bool     `form:"accept_terms"`
		}

		formData := url.Values{
			"plan_id":         {"pro-annual"},
			"billing_cycle":   {"yearly"},
			"payment_method":  {"credit_card"},
			"card_number":     {"4242424242424242"},
			"card_expiry":     {"12/25"},
			"card_cvc":        {"123"},
			"billing_name":    {"John Doe"},
			"billing_email":   {"billing@company.com"},
			"billing_address": {"123 Main St"},
			"billing_city":    {"San Francisco"},
			"billing_state":   {"CA"},
			"billing_zip":     {"94105"},
			"billing_country": {"US"},
			"coupon_code":     {"SAVE20"},
			"addons":          {"extra_storage", "priority_support"},
			"auto_renew":      {"true"},
			"accept_terms":    {"true"},
		}

		req := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result CheckoutForm
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "pro-annual", result.PlanID)
		assert.Equal(t, "yearly", result.BillingCycle)
		assert.Equal(t, []string{"extra_storage", "priority_support"}, result.AddOns)
		assert.True(t, result.AutoRenew)
		assert.True(t, result.AcceptTerms)
	})

	t.Run("project settings form", func(t *testing.T) {
		type ProjectSettings struct {
			ProjectName       string   `form:"project_name"`
			Description       string   `form:"description"`
			Visibility        string   `form:"visibility"`
			DefaultBranch     string   `form:"default_branch"`
			AllowedBranches   []string `form:"allowed_branches"`
			EnableIssues      bool     `form:"enable_issues"`
			EnableWiki        bool     `form:"enable_wiki"`
			EnableDiscussions bool     `form:"enable_discussions"`
			RequirePR         bool     `form:"require_pr"`
			RequireReviews    int      `form:"require_reviews"`
			AutoMerge         bool     `form:"auto_merge"`
			DeleteOnMerge     bool     `form:"delete_branch_on_merge"`
			ProtectedBranches []string `form:"protected_branches"`
			WebhookURL        string   `form:"webhook_url"`
			WebhookSecret     string   `form:"webhook_secret"`
			WebhookEvents     []string `form:"webhook_events"`
			TeamMembers       []string `form:"team_members"`
			AccessLevel       string   `form:"access_level"`
		}

		formData := url.Values{
			"project_name":           {"saaskit"},
			"description":            {"A Go framework for building SaaS applications"},
			"visibility":             {"public"},
			"default_branch":         {"main"},
			"allowed_branches":       {"main,develop,feature/*,hotfix/*"},
			"enable_issues":          {"true"},
			"enable_wiki":            {"true"},
			"enable_discussions":     {"false"},
			"require_pr":             {"true"},
			"require_reviews":        {"2"},
			"auto_merge":             {"true"},
			"delete_branch_on_merge": {"true"},
			"protected_branches":     {"main", "develop"},
			"webhook_url":            {"https://api.company.com/webhooks/github"},
			"webhook_secret":         {"webhook_secret_key"},
			"webhook_events":         {"push", "pull_request", "issues"},
			"team_members":           {"user1@company.com", "user2@company.com"},
			"access_level":           {"write"},
		}

		req := httptest.NewRequest(http.MethodPost, "/projects/settings", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result ProjectSettings
		bindFunc := binder.Form()
		err := bindFunc(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "saaskit", result.ProjectName)
		assert.Equal(t, "public", result.Visibility)
		assert.Equal(t, []string{"main", "develop", "feature/*", "hotfix/*"}, result.AllowedBranches)
		assert.True(t, result.EnableIssues)
		assert.Equal(t, 2, result.RequireReviews)
		assert.Equal(t, []string{"push", "pull_request", "issues"}, result.WebhookEvents)
	})
}
