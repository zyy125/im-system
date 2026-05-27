package dto

type UserRegisterReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserLoginReq struct {
	PublicID uint64 `json:"public_id" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserLoginResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type UserRegisterResp struct {
	PublicID uint64 `json:"public_id"`
}

type UserRefreshReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type UserRefreshResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}
