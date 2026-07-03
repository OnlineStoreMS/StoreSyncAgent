package kdzs

// APIResponse is the common envelope returned by 快递助手 web APIs.
type APIResponse[T any] struct {
	APIName      string `json:"apiName"`
	Data         T      `json:"data"`
	ErrorMessage string `json:"errorMessage"`
	Message      string `json:"message"`
	Result       int    `json:"result"`
	TraceID      string `json:"traceId"`
}

const (
	ResultSuccess        = 100
	ResultPasswordWrong  = 300
	ResultSessionInvalid = 700
	ResultTokenEmpty     = 999
)

// LoginTypePassword uses MD5-hashed password (loginType=1).
const LoginTypePassword = 1

// LoginTypeSMS uses SMS verification code (loginType=2).
const LoginTypeSMS = 2

type LoginRequest struct {
	Mobile    string `json:"mobile"`
	Password  string `json:"password,omitempty"`
	SMSCode   string `json:"smsCode,omitempty"`
	LoginType int    `json:"loginType"`
}

type LoginData struct {
	AccountName       string `json:"accountName"`
	AutoCheck         bool   `json:"autoCheck"`
	BigUser           bool   `json:"bigUser"`
	BindStatus        string `json:"bindStatus"`
	CanTjLogin        bool   `json:"canTjLogin"`
	GrayUser          bool   `json:"grayUser"`
	GrayUser2         bool   `json:"grayUser2"`
	IsRegister        int    `json:"isRegister"`
	JstUser           int    `json:"jstUser"`
	Mobile            string `json:"mobile"`
	RedirectShopName  string `json:"redirectShopName"`
	SubAccountName    string `json:"subAccountName"`
	SubUserID         string `json:"subUserId"`
	SupportStaff      string `json:"supportStaff"`
	Token             string `json:"token"`
	UserID            string `json:"userId"`
}

type BindShop struct {
	ID              int64  `json:"id"`
	UserID          int64  `json:"userId"`
	UserPriID       int64  `json:"userPriId"`
	MallUserID      string `json:"mallUserId"`
	MallUserName    string `json:"mallUserName"`
	Platform        string `json:"platform"`
	BindStatus      int    `json:"bindStatus"`
	BindTime        string `json:"bindTime"`
	Created         string `json:"created"`
	Modified        string `json:"modified"`
	ExpireTime      string `json:"expireTime"`
	TokenExpireDate string `json:"tokenExpireDate"`
	TokenValid      bool   `json:"tokenValid"`
	IsDelete        int    `json:"isDelete"`
	Level           int    `json:"level"`
	OrderCycleType  string `json:"orderCycleType"`
	AppSource       string `json:"appSource"`
	GroupID         string `json:"groupId"`
	GroupName       string `json:"groupName"`
}
