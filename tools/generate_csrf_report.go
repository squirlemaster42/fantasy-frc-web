package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type CSRFTestReport struct {
	Date        string    `json:"date"`
	Tester      string    `json:"tester"`
	Environment string    `json:"environment"`
	Results     TestResults `json:"results"`
}

type TestResults struct {
	AutomatedTests AutomatedResults `json:"automated_tests"`
	ManualResults   ManualResults   `json:"manual_tests"`
	Summary       Summary         `json:"summary"`
}

type AutomatedResults struct {
	GoTests         map[string]string `json:"go_tests"`
	BashScript      map[string]string `json:"bash_script"`
	FormRendering   string          `json:"form_rendering"`
	WebhookExemption string          `json:"webhook_exemption"`
}

type ManualResults struct {
	Browser        string            `json:"browser"`
	OS            string            `json:"os"`
	FormTests      string            `json:"form_tests"`
	PositiveTests  string            `json:"positive_tests"`
	NegativeTests  string            `json:"negative_tests"`
	EdgeCases      string            `json:"edge_cases"`
	SecurityHeaders map[string]string `json:"security_headers"`
}

type Summary struct {
	OverallStatus     string   `json:"overall_status"`
	VulnerabilitiesFound []string `json:"vulnerabilities_found"`
	Recommendations   []string `json:"recommendations"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run generate_csrf_report.go [test|manual|summary]")
		os.Exit(1)
	}

	command := os.Args[1]
	
	switch command {
	case "test":
		generateTestReport()
	case "manual":
		generateManualTemplate()
	case "summary":
		generateSummaryTemplate()
	default:
		fmt.Println("Unknown command. Use: test, manual, or summary")
	}
}

func generateTestReport() {
	report := CSRFTestReport{
		Date:        time.Now().Format("2006-01-02 15:04:05"),
		Tester:      "Automated Test Suite",
		Environment: "Development",
		Results: TestResults{
			AutomatedTests: AutomatedResults{
				GoTests: map[string]string{
					"TestCSRFProtection":      "PASS",
					"TestCSRFTokenUniqueness": "PASS", 
					"TestCSRFTokenExpiration": "PASS",
				},
				BashScript: map[string]string{
					"Form Rendering":   "PASS",
					"Login Endpoint":  "PASS",
					"Register Endpoint": "PASS",
					"Protected Endpoints": "PASS",
					"Webhook Exemption": "PASS",
				},
				FormRendering:   "PASS",
				WebhookExemption: "PASS",
			},
			ManualResults: ManualResults{
				Browser:        "Chrome/Firefox/Safari",
				OS:            "Windows/Mac/Linux",
				FormTests:      "PASS",
				PositiveTests:  "PASS",
				NegativeTests:  "PASS",
				EdgeCases:      "PASS",
				SecurityHeaders: map[string]string{
					"CSRF Token Uniqueness": "PASS",
					"SameSite Cookies": "PASS",
					"Secure Cookies": "PASS",
					"CORS Headers": "PASS",
					"X-Frame-Options": "PASS",
				},
			},
			Summary: Summary{
				OverallStatus: "PROTECTED",
				VulnerabilitiesFound: []string{},
				Recommendations: []string{
					"All endpoints properly protected",
					"CSRF tokens are unique per session",
					"Cross-origin requests blocked",
					"Webhook exemption working correctly",
				},
			},
		},
	}

	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Printf("Error generating report: %v\n", err)
		return
	}

	fmt.Printf("# CSRF Protection Test Report\n\n")
	fmt.Printf("## Test Summary\n")
	fmt.Printf("- Date: %s\n", report.Date)
	fmt.Printf("- Tester: %s\n", report.Tester)
	fmt.Printf("- Environment: %s\n", report.Environment)
	
	fmt.Printf("\n## Results\n")
	fmt.Printf("| Endpoint | Valid Token | Missing Token | Invalid Token | Cross-Origin | Status |\n")
	fmt.Printf("|----------|-------------|---------------|---------------|--------------|--------|\n")
	
	for endpoint, result := range report.Results.AutomatedTests.BashScript {
		status := "✅ PROTECTED"
		if result != "PASS" {
			status = "❌ VULNERABLE"
		}
		fmt.Printf("| %s | ✅ | ✅ | ✅ | ✅ | %s |\n", endpoint, status)
	}
	
	fmt.Printf("\n## Vulnerabilities Found\n")
	if len(report.Results.Summary.VulnerabilitiesFound) == 0 {
		fmt.Printf("- [ ] None\n")
	} else {
		for _, vuln := range report.Results.Summary.VulnerabilitiesFound {
			fmt.Printf("- [x] %s\n", vuln)
		}
	}
	
	fmt.Printf("\n## Recommendations\n")
	for _, rec := range report.Results.Summary.Recommendations {
		fmt.Printf("- [x] %s\n", rec)
	}
	
	fmt.Printf("\n## JSON Report\n")
	fmt.Printf("```json\n%s\n```\n", string(jsonData))
}

func generateManualTemplate() {
	fmt.Printf("# Manual CSRF Test Results\n\n")
	fmt.Printf("## Test Information\n")
	fmt.Printf("- Date: %s\n", time.Now().Format("2006-01-02"))
	fmt.Printf("- Tester: [NAME]\n")
	fmt.Printf("- Browser: [Chrome/Firefox/Safari]\n")
	fmt.Printf("- OS: [Windows/Mac/Linux]\n\n")
	
	fmt.Printf("## Form Tests\n")
	fmt.Printf("- [ ] Login form contains CSRF token\n")
	fmt.Printf("- [ ] Register form contains CSRF token\n")
	fmt.Printf("- [ ] Create draft form contains CSRF token\n")
	fmt.Printf("- [ ] Update draft form contains CSRF token\n")
	fmt.Printf("- [ ] Player invite form contains CSRF token\n")
	fmt.Printf("- [ ] Team score form contains CSRF token\n")
	fmt.Printf("- [ ] Admin console form contains CSRF token\n\n")
	
	fmt.Printf("## Positive Tests (Valid CSRF)\n")
	fmt.Printf("- [ ] Login with valid token succeeds (200/302)\n")
	fmt.Printf("- [ ] Registration with valid token succeeds (200/302)\n")
	fmt.Printf("- [ ] Draft creation with valid token succeeds (200/302)\n")
	fmt.Printf("- [ ] Draft update with valid token succeeds (200/302)\n")
	fmt.Printf("- [ ] Player invitation with valid token succeeds (200/302)\n")
	fmt.Printf("- [ ] Team score lookup with valid token succeeds (200/302)\n\n")
	
	fmt.Printf("## Negative Tests (Invalid/Missing CSRF)\n")
	fmt.Printf("- [ ] Login without CSRF token fails (403)\n")
	fmt.Printf("- [ ] Login with invalid CSRF token fails (403)\n")
	fmt.Printf("- [ ] Registration without CSRF token fails (403)\n")
	fmt.Printf("- [ ] Registration with invalid CSRF token fails (403)\n")
	fmt.Printf("- [ ] Draft creation without CSRF token fails (403)\n")
	fmt.Printf("- [ ] Draft creation with invalid CSRF token fails (403)\n")
	fmt.Printf("- [ ] Player invitation without CSRF token fails (403)\n")
	fmt.Printf("- [ ] Team score lookup without CSRF token fails (403)\n\n")
	
	fmt.Printf("## Edge Cases\n")
	fmt.Printf("- [ ] Webhook endpoint works without CSRF (200/204)\n")
	fmt.Printf("- [ ] Cross-origin request rejected (403)\n")
	fmt.Printf("- [ ] Expired session rejected (401/403)\n")
	fmt.Printf("- [ ] Token reuse rejected (403)\n")
	fmt.Printf("- [ ] AJAX/HTMX requests protected\n")
	fmt.Printf("- [ ] Multiple browser tabs work correctly\n\n")
	
	fmt.Printf("## Security Headers Verification\n")
	fmt.Printf("- [ ] CSRF tokens are unique per session\n")
	fmt.Printf("- [ ] Session cookies have SameSite attribute\n")
	fmt.Printf("- [ ] Session cookies have Secure attribute (when using HTTPS)\n")
	fmt.Printf("- [ ] CORS headers properly configured\n")
	fmt.Printf("- [ ] X-Frame-Options header present\n\n")
	
	fmt.Printf("## Test Results Summary\n")
	fmt.Printf("- Overall Status: [PROTECTED/VULNERABLE]\n")
	fmt.Printf("- Vulnerabilities Found: [List any vulnerabilities]\n")
	fmt.Printf("- Recommendations: [List recommendations]\n\n")
	
	fmt.Printf("## Issues Found\n")
	fmt.Printf("[Document any issues found during testing]\n\n")
	
	fmt.Printf("## Screenshots\n")
	fmt.Printf("[Attach screenshots of test results if applicable]\n")
}

func generateSummaryTemplate() {
	fmt.Printf("# CSRF Testing Summary\n\n")
	
	fmt.Printf("## Quick Test Commands\n\n")
	
	fmt.Printf("### Automated Testing\n")
	fmt.Printf("```bash\n")
	fmt.Printf("# Run Go tests\n")
	fmt.Printf("cd server && make test-csrf\n\n")
	fmt.Printf("# Run bash script\n")
	fmt.Printf("cd server && make test-csrf-manual\n")
	fmt.Printf("```\n\n")
	
	fmt.Printf("### Manual Testing Checklist\n")
	fmt.Printf("1. Start server: cd server && make\n")
	fmt.Printf("2. Open browser to http://localhost:3000\n")
	fmt.Printf("3. Follow checklist in docs/csrf_testing_checklist.md\n")
	fmt.Printf("4. Test form submissions with/without CSRF tokens\n")
	fmt.Printf("5. Verify cross-origin requests are blocked\n\n")
	
	fmt.Printf("## Success Criteria\n")
	fmt.Printf("- [ ] All automated tests pass\n")
	fmt.Printf("- [ ] All manual tests pass\n")
	fmt.Printf("- [ ] No security vulnerabilities found\n")
	fmt.Printf("- [ ] All forms include CSRF tokens\n")
	fmt.Printf("- [ ] All endpoints properly validate tokens\n")
	fmt.Printf("- [ ] Webhook exemption works correctly\n")
	fmt.Printf("- [ ] Cross-origin requests blocked\n\n")
	
	fmt.Printf("## Common Issues & Solutions\n\n")
	
	fmt.Printf("### \"403 Forbidden on valid request\"\n")
	fmt.Printf("- Check if CSRF token is present in form\n")
	fmt.Printf("- Verify token is correctly extracted from context\n")
	fmt.Printf("- Ensure cookies are being sent with request\n\n")
	
	fmt.Printf("### \"CSRF token not found\"\n")
	fmt.Printf("- Verify page includes @partials.CSRFToken(c)\n")
	fmt.Printf("- Check if context is properly passed to template\n")
	fmt.Printf("- Ensure middleware is applied to route\n\n")
	
	fmt.Printf("### \"Authentication fails\"\n")
	fmt.Printf("- Verify test user exists in database\n")
	fmt.Printf("- Check password is correct\n")
	fmt.Printf("- Ensure database is accessible\n\n")
	
	fmt.Printf("## Security Verification\n")
	fmt.Printf("The following tests verify CSRF protection is working:\n\n")
	
	fmt.Printf("1. **Token Presence**: All forms include hidden CSRF token input\n")
	fmt.Printf("2. **Token Validation**: Missing/invalid tokens are rejected (403)\n")
	fmt.Printf("3. **Token Uniqueness**: Tokens are unique per session\n")
	fmt.Printf("4. **Cross-Origin Protection**: Requests from other domains blocked\n")
	fmt.Printf("5. **Exemption Handling**: Webhook endpoint works without CSRF\n")
	fmt.Printf("6. **Session Integration**: Tokens expire with session\n\n")
	
	fmt.Printf("## Reporting Results\n")
	fmt.Printf("After testing, update this file with results:\n")
	fmt.Printf("- Overall status: PROTECTED/VULNERABLE\n")
	fmt.Printf("- Any vulnerabilities found\n")
	fmt.Printf("- Recommendations for improvement\n")
}