package openlibrary

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"book-metaverse/backend/internal/domain"
)

const (
	// Source is books.source for imported works.
	Source = "openlibrary"

	maxTitleRunes       = 500
	maxNameRunes        = 200
	maxAuthors          = 20
	maxCategories       = 5
	maxCategoryRunes    = 50
	maxDescriptionRunes = 10000
	maxPublisherRunes   = 200
	maxPageCount        = 100000
	maxISBNsPerWork     = 200
)

// WorkIDPattern is an Open Library work ID (books.openlibrary_id).
var WorkIDPattern = regexp.MustCompile(`^OL[0-9]+W$`)

// doc is one /search.json result with the fields this client requests.
type doc struct {
	Key          string   `json:"key"`
	Title        string   `json:"title"`
	Subtitle     string   `json:"subtitle"`
	AuthorName   []string `json:"author_name"`
	FirstPublish int      `json:"first_publish_year"`
	PagesMedian  int      `json:"number_of_pages_median"`
	CoverID      int64    `json:"cover_i"`
	Subject      []string `json:"subject"`
	ISBN         []string `json:"isbn"`
	Editions     struct {
		Docs []struct {
			ISBN      []string `json:"isbn"`
			Publisher []string `json:"publisher"`
		} `json:"docs"`
	} `json:"editions"`
}

// searchFields is the fields= list; editions.* returns the edition that best
// matches the query, which supplies the representative ISBN and publisher.
const searchFields = "key,title,subtitle,author_name,first_publish_year,number_of_pages_median,cover_i,subject,isbn," +
	"editions,editions.isbn,editions.publisher"

// normalize turns a search doc into an ExternalBook. ok is false for a doc
// without a usable work key or title, which is then skipped.
func (c *Client) normalize(d doc) (domain.ExternalBook, bool) {
	id := strings.TrimPrefix(d.Key, "/works/")
	title := cleanText(d.Title, maxTitleRunes)
	if !WorkIDPattern.MatchString(id) || title == "" {
		return domain.ExternalBook{}, false
	}
	b := domain.ExternalBook{
		Source:     Source,
		ExternalID: id,
		Title:      title,
		Subtitle:   optional(cleanText(d.Subtitle, maxTitleRunes)),
		Authors:    names(d.AuthorName, maxAuthors),
		Categories: Categories(d.Subject),
		InfoLink:   c.siteURL + "/works/" + id,
		ISBN13s:    isbn13s(d.ISBN),
	}
	if d.FirstPublish > 0 && d.FirstPublish <= time.Now().Year()+1 {
		year := strconv.Itoa(d.FirstPublish)
		b.PublishedDate = &year
	}
	if d.PagesMedian > 0 && d.PagesMedian <= maxPageCount {
		n := int32(d.PagesMedian)
		b.PageCount = &n
	}
	if d.CoverID > 0 {
		cover := c.coversURL + "/b/id/" + strconv.FormatInt(d.CoverID, 10) + "-M.jpg"
		b.CoverURL = &cover
	}
	if len(d.Editions.Docs) > 0 {
		best := d.Editions.Docs[0]
		if isbns := isbn13s(best.ISBN); len(isbns) > 0 {
			b.ISBN13 = &isbns[0]
		}
		if len(best.Publisher) > 0 {
			b.Publisher = optional(cleanText(best.Publisher[0], maxPublisherRunes))
		}
	}
	// The representative ISBN belongs to the work too.
	if b.ISBN13 != nil && !contains(b.ISBN13s, *b.ISBN13) {
		b.ISBN13s = append(b.ISBN13s, *b.ISBN13)
	}
	return b, true
}

// cleanText collapses whitespace, drops control characters, and truncates.
func cleanText(s string, max int) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' {
			return ' '
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) > max {
		s = strings.TrimSpace(string([]rune(s)[:max]))
	}
	return s
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// names trims, drops empties, and deduplicates case-insensitively.
func names(in []string, max int) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, n := range in {
		n = cleanText(n, maxNameRunes)
		key := strings.ToLower(n)
		if n == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, n)
		if len(out) == max {
			break
		}
	}
	return out
}

// noisySubject matches Open Library subjects that are not genres: library
// and lending tags, list tags (nyt:...), reading levels, translations.
var noisySubject = regexp.MustCompile(`(?i)accessible book|protected daisy|in library|lending library|overdrive|large type|` +
	`new york times|reading level|translation|staff picks|award|open library|long now|textbooks|juvenile literature`)

// Categories keeps up to five genre-like subjects. Open Library subjects mix
// genres with catalog headings ("Fiction, science fiction, general"), list
// tags ("nyt:...=2021-11-07"), and place names; headings with a comma, colon,
// equals sign, or parentheses, long phrases, and known noise are dropped.
func Categories(subjects []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, s := range subjects {
		s = cleanText(s, 1000)
		if s == "" || utf8.RuneCountInString(s) > maxCategoryRunes || strings.ContainsAny(s, ",:=()[]/") ||
			noisySubject.MatchString(s) || len(strings.Fields(s)) > 4 {
			continue
		}
		// "Science fiction" and "Science-fiction" are one category.
		key := strings.ToLower(strings.ReplaceAll(s, "-", " "))
		if seen[key] {
			continue
		}
		seen[key] = true
		r, size := utf8.DecodeRuneInString(s)
		out = append(out, string(unicode.ToUpper(r))+s[size:])
		if len(out) == maxCategories {
			break
		}
	}
	return out
}

// isbn13s returns the ISBN-13s in the list, converting ISBN-10s, without
// duplicates and capped (a popular work lists hundreds of editions).
func isbn13s(in []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, raw := range in {
		isbn, ok := ISBN13(raw)
		if !ok || seen[isbn] {
			continue
		}
		seen[isbn] = true
		out = append(out, isbn)
		if len(out) == maxISBNsPerWork {
			break
		}
	}
	return out
}

// ISBN13 validates an ISBN-13 or converts an ISBN-10 (both with check digits);
// hyphens and spaces are ignored.
func ISBN13(raw string) (string, bool) {
	s := strings.ToUpper(strings.NewReplacer("-", "", " ", "").Replace(raw))
	switch len(s) {
	case 13:
		if !allDigits(s) || isbn13Check(s[:12]) != s[12] {
			return "", false
		}
		return s, true
	case 10:
		if !allDigits(s[:9]) || (s[9] != 'X' && (s[9] < '0' || s[9] > '9')) {
			return "", false
		}
		sum := 0
		for i := 0; i < 9; i++ {
			sum += int(s[i]-'0') * (10 - i)
		}
		check := 10
		if s[9] != 'X' {
			check = int(s[9] - '0')
		}
		if (sum+check)%11 != 0 {
			return "", false
		}
		base := "978" + s[:9]
		return base + string(isbn13Check(base)), true
	}
	return "", false
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func isbn13Check(first12 string) byte {
	sum := 0
	for i := 0; i < 12; i++ {
		d := int(first12[i] - '0')
		if i%2 == 1 {
			d *= 3
		}
		sum += d
	}
	return byte('0' + (10-sum%10)%10)
}

// description reads a work's description, which is a string or
// {"type": "/type/text", "value": "..."}. Open Library descriptions often end
// with a "----------" section of related links and markdown link references;
// both are removed.
func description(raw any) *string {
	var s string
	switch v := raw.(type) {
	case string:
		s = v
	case map[string]any:
		s, _ = v["value"].(string)
	}
	if i := strings.Index(s, "\n----------"); i >= 0 {
		s = s[:i]
	}
	lines := []string{}
	for _, line := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		if markdownRef.MatchString(line) {
			continue
		}
		lines = append(lines, strings.TrimRight(line, " \t"))
	}
	s = strings.TrimSpace(multiBlank.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' {
			return -1
		}
		return r
	}, s)
	if utf8.RuneCountInString(s) > maxDescriptionRunes {
		s = strings.TrimSpace(string([]rune(s)[:maxDescriptionRunes]))
	}
	return optional(s)
}

var (
	markdownRef = regexp.MustCompile(`^\s*\[[0-9]+\]:\s*\S+`)
	multiBlank  = regexp.MustCompile(`\n{3,}`)
)
