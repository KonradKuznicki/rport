package oauth

import (
	"errors"
	"net/http"
	"plugin"
	"time"

	"github.com/cloudradar-monitoring/rport/plus/validator"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

var (
	ErrMissingConfig       = errors.New("no config to validate")
	ErrMissingProvider     = errors.New("missing provider")
	ErrMissingAuthorizeURL = errors.New("missing authorize_url")
	ErrMissingTokenURL     = errors.New("missing token_url")
	ErrMissingRedirectURI  = errors.New("missing redirect_uri")
	ErrMissingClientID     = errors.New("missing client_id")
	ErrMissingClientSecret = errors.New("missing client_secret")
)

type LoginInfo struct {
	LoginMsg     string    `json:"message"`
	AuthorizeURL string    `json:"authorize_url"`
	LoginURI     string    `json:"login_uri"`
	State        string    `json:"state"`
	Expiry       time.Time `json:"expiry"`
}

// CapabilityEx represents the functional interface provided by the OAuth capability
type CapabilityEx interface {
	ValidateConfig() (err error)
	GetOAuthLoginInfo() (loginInfo *LoginInfo, err error)
	PerformAuthCodeExchange(r *http.Request) (token string, username string, err error)
	GetPermittedUser(r *http.Request, token string) (username string, err error)
}

// Config is the OAuth capability config, as loaded from the rportd config file
type Config struct {
	Provider             string `mapstructure:"provider"`
	BaseAuthorizeURL     string `mapstructure:"authorize_url"`
	TokenURL             string `mapstructure:"token_url"`
	RedirectURI          string `mapstructure:"redirect_uri"`
	ClientID             string `mapstructure:"client_id"`
	ClientSecret         string `mapstructure:"client_secret"`
	RequiredOrganization string `mapstructure:"required_organization"`
	PermittedUserList    bool   `mapstructure:"permitted_user_list"`

	// currently only used by the Auth0 provider
	JWKSURL       string `mapstructure:"jwks_url"`
	RoleClaim     string `mapstructure:"role_claim"`
	RequiredRole  string `mapstructure:"required_role"`
	UsernameClaim string `mapstructure:"username_claim"`
}

const (
	InitOAuthCapabilityEx  = "InitOAuthCapabilityEx"
	GitHubOAuthProvider    = "github"
	MicrosoftOAuthProvider = "microsoft"
	Auth0OAuthProvider     = "auth0"

	DefaultLoginURI = "/oauth/exchangecode"
)

// Capability is used by rportd to maintain loaded info about the plugin's
// oauth capability
type Capability struct {
	Provider CapabilityEx

	Config *Config
	Logger *logger.Logger
}

// GetInitFuncName gets the name of the capability init func
func (cap *Capability) GetInitFuncName() (name string) {
	return InitOAuthCapabilityEx
}

// SetProvider invokes the capability init func in the plugin and saves
// the returned capability provider interface. This interface provides
// the functions of the capability.
func (cap *Capability) SetProvider(initFn plugin.Symbol) {
	fn := initFn.(func(cap *Capability) (capProvider CapabilityEx))
	cap.Provider = fn(cap)
}

// GetOAuthCapabilityEx returns the interface to the capability functions
func (cap *Capability) GetOAuthCapabilityEx() (capEx CapabilityEx) {
	return cap.Provider
}

// GetConfigValidator returns a validator interface that can be called to
// validate the capability config
func (cap *Capability) GetConfigValidator() (v validator.Validator) {
	return cap.Provider
}