// Copyright (c) 2026 Madalin Gabriel Ignisca <hi@madalin.me>
// Copyright (c) 2026 Vlah Software House SRL <contact@vlah.sh>
// All rights reserved. See LICENSE for details.

package handlers

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// subdomainPattern allows only lowercase alphanumeric characters and hyphens,
// must start and end with alphanumeric, 3-63 chars (DNS label rules).
var subdomainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$`)

// blockedSubdomains contains reserved and offensive subdomain names that
// cannot be used for tenant subdomains. All entries must be lowercase.
var blockedSubdomains = map[string]bool{
	// Infrastructure & protocol reserved names.
	"www":        true,
	"www1":       true,
	"www2":       true,
	"www3":       true,
	"mail":       true,
	"smtp":       true,
	"imap":       true,
	"pop":        true,
	"pop3":       true,
	"email":      true,
	"webmail":    true,
	"ftp":        true,
	"sftp":       true,
	"ssh":        true,
	"ns":         true,
	"ns1":        true,
	"ns2":        true,
	"ns3":        true,
	"dns":        true,
	"mx":         true,
	"vpn":        true,
	"proxy":      true,
	"cdn":        true,
	"api":        true,
	"api-v1":     true,
	"api-v2":     true,
	"graphql":    true,
	"grpc":       true,
	"ws":         true,
	"wss":        true,
	"tcp":        true,

	// Platform & system names.
	"admin":      true,
	"dashboard":  true,
	"app":        true,
	"login":      true,
	"auth":       true,
	"oauth":      true,
	"sso":        true,
	"signup":     true,
	"register":   true,
	"account":    true,
	"accounts":   true,
	"profile":    true,
	"settings":   true,
	"config":     true,
	"panel":      true,
	"console":    true,
	"portal":     true,
	"support":    true,
	"help":       true,
	"docs":       true,
	"status":     true,
	"health":     true,
	"monitor":    true,
	"metrics":    true,
	"staging":    true,
	"dev":        true,
	"test":       true,
	"demo":       true,
	"sandbox":    true,
	"beta":       true,
	"alpha":      true,
	"preview":    true,
	"internal":   true,
	"private":    true,
	"public":     true,
	"static":     true,
	"assets":     true,
	"media":      true,
	"images":     true,
	"img":        true,
	"files":      true,
	"uploads":    true,
	"download":   true,
	"downloads":  true,
	"blog":       true,
	"shop":       true,
	"store":      true,
	"billing":    true,
	"payment":    true,
	"pay":        true,
	"checkout":   true,
	"search":     true,
	"calendar":   true,
	"forum":      true,
	"community":  true,
	"wiki":       true,
	"git":        true,
	"svn":        true,
	"repo":       true,
	"ci":         true,
	"cd":         true,
	"jenkins":    true,
	"deploy":     true,
	"root":       true,
	"system":     true,
	"default":    true,
	"localhost":  true,
	"local":      true,
	"null":       true,
	"undefined":  true,
	"info":       true,
	"about":      true,
	"contact":    true,
	"legal":      true,
	"privacy":    true,
	"terms":      true,
	"tos":        true,

	// Brand protection.
	"smartpress": true,
	"yaaicms":    true,

	// Offensive / vulgar terms.
	"ass":         true,
	"asshole":     true,
	"bastard":     true,
	"bitch":       true,
	"bollocks":    true,
	"cock":        true,
	"crap":        true,
	"cunt":        true,
	"damn":        true,
	"dick":        true,
	"douche":      true,
	"fag":         true,
	"faggot":      true,
	"fuck":        true,
	"fucker":      true,
	"fucking":     true,
	"goddamn":     true,
	"hell":        true,
	"jerk":        true,
	"kike":        true,
	"milf":        true,
	"motherfucker": true,
	"nazi":        true,
	"nigga":       true,
	"nigger":      true,
	"penis":       true,
	"piss":        true,
	"porn":        true,
	"porno":       true,
	"pussy":       true,
	"rape":        true,
	"rapist":      true,
	"retard":      true,
	"retarded":    true,
	"sex":         true,
	"sexy":        true,
	"shit":        true,
	"shitty":      true,
	"slut":        true,
	"spic":        true,
	"tits":        true,
	"twat":        true,
	"vagina":      true,
	"whore":       true,
	"wanker":      true,
	"xxx":         true,
}

// validateSubdomain checks that a subdomain is well-formed and not reserved or
// offensive. Returns an error message or empty string if valid.
func validateSubdomain(subdomain string) string {
	if len(subdomain) < 3 {
		return "Subdomain must be at least 3 characters."
	}
	if !subdomainPattern.MatchString(subdomain) {
		return "Subdomain may only contain lowercase letters, numbers, and hyphens, and must start and end with a letter or number."
	}
	if blockedSubdomains[subdomain] {
		return fmt.Sprintf("The subdomain %q is reserved and cannot be used.", subdomain)
	}
	// Also check if the subdomain contains an offensive word as a substring.
	for word := range blockedSubdomains {
		// Only check substring matches for offensive terms (skip short infra names).
		if len(word) >= 4 && strings.Contains(subdomain, word) {
			return fmt.Sprintf("The subdomain contains a restricted term and cannot be used.")
		}
	}
	return ""
}

// Validation limits for content and template fields.
const (
	maxTitleLen       = 300
	maxSlugLen        = 300
	maxBodyLen        = 100_000
	maxExcerptLen     = 1_000
	maxMetaDescLen    = 500
	maxMetaKeywordLen = 500
	maxTemplateNameLen = 200
	maxTemplateHTMLLen = 500_000
)

// validateContent checks content form inputs and returns the first error found.
func validateContent(title, slug, body string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "Title is required."
	}
	if utf8.RuneCountInString(title) > maxTitleLen {
		return "Title is too long (max 300 characters)."
	}
	if utf8.RuneCountInString(slug) > maxSlugLen {
		return "Slug is too long (max 300 characters)."
	}
	if utf8.RuneCountInString(body) > maxBodyLen {
		return "Body is too long (max 100,000 characters)."
	}
	return ""
}

// validateMetadata checks optional SEO metadata fields.
func validateMetadata(excerpt, metaDesc, metaKw string) string {
	if utf8.RuneCountInString(excerpt) > maxExcerptLen {
		return "Excerpt is too long (max 1,000 characters)."
	}
	if utf8.RuneCountInString(metaDesc) > maxMetaDescLen {
		return "Meta description is too long (max 500 characters)."
	}
	if utf8.RuneCountInString(metaKw) > maxMetaKeywordLen {
		return "Meta keywords are too long (max 500 characters)."
	}
	return ""
}

// validateTemplate checks template form inputs and returns the first error found.
func validateTemplate(name, htmlContent string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Template name is required."
	}
	if utf8.RuneCountInString(name) > maxTemplateNameLen {
		return "Template name is too long (max 200 characters)."
	}
	htmlContent = strings.TrimSpace(htmlContent)
	if htmlContent == "" {
		return "Template HTML content is required."
	}
	if utf8.RuneCountInString(htmlContent) > maxTemplateHTMLLen {
		return "Template HTML content is too long (max 500,000 characters)."
	}
	return ""
}
