package ldapserver

import (
	"fmt"
	"regexp"

	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
	ldap "github.com/vjeantet/ldapserver"
)

const defaultBaseDN = "dc=example,dc=org"

var serialRegex = regexp.MustCompile(`(?i)serialNumber=(%\{[^}]+\}|[^\(\)\s=]+)`)

// newRouteMux は Bind / Extended(WhoAmI) / Search のハンドラーを登録したルーターを返す
func newRouteMux() *ldap.RouteMux {
	routes := ldap.NewRouteMux()
	routes.Bind(handleSimpleBind).AuthenticationChoice("simple")
	routes.Extended(handleWhoAmI).RequestName(ldap.NoticeOfWhoAmI).Label("Ext - WhoAmI")
	routes.Search(handleSearch)

	return routes
}

func handleSimpleBind(w ldap.ResponseWriter, m *ldap.Message) {
	r := m.GetBindRequest()
	bindDN := string(r.Name())
	res := ldap.NewBindResponse(ldap.LDAPResultSuccess)

	switch bindDN {
	case "", "cn=admin," + defaultBaseDN, "myLogin", "uid=myLogin," + defaultBaseDN:
		logger.Info(fmt.Sprintf("[Simple Bind] Success for DN: %s", bindDN))
		m.Client.SetData(bindDN)
		w.Write(res)

		return
	}

	logger.Warn(fmt.Sprintf("[Simple Bind] Failed for DN: %s", bindDN))
	res.SetResultCode(ldap.LDAPResultInvalidCredentials)
	res.SetDiagnosticMessage("invalid credentials")
	w.Write(res)
}

func handleSearch(w ldap.ResponseWriter, m *ldap.Message) {
	r := m.GetSearchRequest()
	filter := r.FilterString()
	baseDN := string(r.BaseObject())

	boundDN, _ := m.Client.GetData().(string)
	logger.Info(fmt.Sprintf("[Search] Request received from '%s' - BaseDN: %s, Filter: %s", boundDN, baseDN, filter))

	serialID := extractSerialID(m, filter)

	select {
	case <-m.Done:
		logger.Info(fmt.Sprintf("[Search] Operation canceled for message ID: %d", m.MessageID()))
		w.Write(ldap.NewSearchResultDoneResponse(ldap.LDAPResultCanceled))

		return
	default:
	}

	if serialID != "" {
		logger.Info(fmt.Sprintf("[Search] Checking certificate status for key: %s", serialID))

		// TODO: 証明書の有効性を確認する
		isValidCert := true
		foundDeviceID := serialID

		if isValidCert {
			logger.Info(fmt.Sprintf("[Search] Valid certificate found for key: %s", foundDeviceID))

			if baseDN == "" {
				baseDN = defaultBaseDN
			}

			w.Write(ldap.NewSearchResultEntry("uid=" + foundDeviceID + "," + baseDN))
		}
	}

	w.Write(ldap.NewSearchResultDoneResponse(ldap.LDAPResultSuccess))
}

// extractSerialID は mTLS のクライアント証明書、なければ検索フィルターからシリアル番号を取り出す
func extractSerialID(m *ldap.Message, filter string) string {
	if tlsConn := tlsConnOf(m.Client.GetConn()); tlsConn != nil {
		state := tlsConn.ConnectionState()
		if state.HandshakeComplete && len(state.PeerCertificates) > 0 {
			serialID := state.PeerCertificates[0].SerialNumber.String()
			logger.Info(fmt.Sprintf("[Search] Extracted serialNumber from mTLS Peer Certificate: %s", serialID))

			return serialID
		}
	}

	if matches := serialRegex.FindStringSubmatch(filter); len(matches) > 1 {
		logger.Info(fmt.Sprintf("[Search] Extracted serialNumber from Search Filter: %s", matches[1]))

		return matches[1]
	}

	return ""
}

func handleWhoAmI(w ldap.ResponseWriter, _ *ldap.Message) {
	w.Write(ldap.NewExtendedResponse(ldap.LDAPResultSuccess))
}
