package middleware

import (
	"net/http"
	"strings"

	"storesyncagent/internal/config"
	"storesyncagent/internal/pkg/authcontext"
	jwtmgr "storesyncagent/internal/pkg/jwt"
	"storesyncagent/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func AdminAuth(cfg *config.AuthConfig, jwt *jwtmgr.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			response.Fail(c, http.StatusUnauthorized, "认证未启用")
			c.Abort()
			return
		}
		if jwt == nil {
			response.Fail(c, http.StatusUnauthorized, "JWT 未配置")
			c.Abort()
			return
		}
		token := ""
		auth := c.GetHeader("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		}
		if token == "" {
			if ck, err := c.Request.Cookie("uc_access"); err == nil && ck != nil {
				token = strings.TrimSpace(ck.Value)
			}
		}
		if token == "" {
			response.Fail(c, http.StatusUnauthorized, "请先登录")
			c.Abort()
			return
		}
		claims, err := jwt.ParseAccess(token)
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, "登录已过期，请重新登录")
			c.Abort()
			return
		}
		if claims.TenantID == 0 {
			response.Fail(c, http.StatusUnauthorized, "请选择租户")
			c.Abort()
			return
		}
		c.Set(authcontext.ContextClaims, claims)
		c.Set(authcontext.ContextTenant, claims.TenantID)
		c.Set(authcontext.ContextUser, claims.UserID)
		c.Next()
	}
}
