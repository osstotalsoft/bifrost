package openidclient

type Config struct {
	Issuer                                string   `json:"issuer"`
	JwksUri                               string   `json:"jwks_uri"`
	AuthorizationEndpoint                 string   `json:"authorization_endpoint"`
	TokenEndpoint                         string   `json:"token_endpoint"`
	UserinfoEndpoint                      string   `json:"userinfo_endpoint"`
	EndSessionEndpoint                    string   `json:"end_session_endpoint"`
	CheckSessionIframe                    string   `json:"check_session_iframe"`
	RevocationEndpoint                    string   `json:"revocation_endpoint"`
	IntrospectionEndpoint                 string   `json:"introspection_endpoint"`
	FrontchannelLogoutSupported           string   `json:"frontchannel_logout_supported"`
	Frontchannel_logout_session_supported string   `json:"frontchannel_logout_session_supported"`
	backchannel_logout_supported          string   `json:"backchannel_logout_supported"`
	backchannel_logout_session_supported  string   `json:"backchannel_logout_session_supported"`
	scopes_supported                      []string `json:"scopes_supported"`
	claims_supported                      []string `json:"claims_supported"`
	grant_types_supported                 []string `json:"grant_types_supported"`
	response_types_supported              []string `json:"response_types_supported"`
	response_modes_supported              []string `json:"response_modes_supported"`
	token_endpoint_auth_methods_supported []string `json:"token_endpoint_auth_methods_supported"`
	subject_types_supported               []string `json:"subject_types_supported"`
	id_token_signing_alg_values_supported []string `json:"id_token_signing_alg_values_supported"`
	code_challenge_methods_supported      []string `json:"code_challenge_methods_supported"`
}
