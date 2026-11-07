package libcore

import (
	"strings"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
)

// Sources of headless rules for the legacy geoip.db / geosite.db files bundled
// with the app. Variables so tests can stub them.
var (
	geoIPRules   = geoIPHeadlessRules
	geoSiteRules = geoSiteHeadlessRules
)

// rewriteGeoRuleSets turns the app's pseudo rule-sets
//
//	{"type": "local", "tag": "x", "path": "geoip:cn" | "geosite:google"}
//
// into stock inline rule-sets, so sing-box needs no patch to understand them.
func rewriteGeoRuleSets(ruleSets []option.RuleSet) error {
	for i := range ruleSets {
		rs := &ruleSets[i]
		if rs.Type != C.RuleSetTypeLocal {
			continue
		}
		var (
			name  string
			rules []option.HeadlessRule
			err   error
		)
		switch path := rs.LocalOptions.Path; {
		case strings.HasPrefix(path, "geoip:"):
			name = strings.TrimPrefix(path, "geoip:")
			rules, err = geoIPRules(name)
		case strings.HasPrefix(path, "geosite:"):
			name = strings.TrimPrefix(path, "geosite:")
			rules, err = geoSiteRules(name)
		default:
			continue
		}
		if err != nil {
			return E.Cause(err, "load ", rs.LocalOptions.Path, " for rule-set ", strings.Join(rs.Tag, ","))
		}
		if len(rules) == 0 {
			return E.New("empty rule-set: ", rs.LocalOptions.Path)
		}
		rs.Type = C.RuleSetTypeInline
		rs.Format = ""
		rs.LocalOptions = option.LocalRuleSet{}
		rs.InlineOptions = option.PlainRuleSet{Rules: rules}
	}
	return nil
}

