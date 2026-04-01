package relay

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

var signingKey []byte

func init() {
	signingKey = make([]byte, 32)
	if _, err := rand.Read(signingKey); err != nil {
		panic(err)
	}
}

func generatePassword() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)[:16]
}

func signCookie(value string) string {
	mac := hmac.New(sha256.New, signingKey)
	mac.Write([]byte(value))
	sig := hex.EncodeToString(mac.Sum(nil))
	return value + "." + sig
}

func verifyCookie(signed string) bool {
	for i := len(signed) - 1; i >= 0; i-- {
		if signed[i] == '.' {
			value := signed[:i]
			sig := signed[i+1:]
			mac := hmac.New(sha256.New, signingKey)
			mac.Write([]byte(value))
			expected := hex.EncodeToString(mac.Sum(nil))
			return hmac.Equal([]byte(sig), []byte(expected))
		}
	}
	return false
}

func (r *Relay) checkUIPassword(w http.ResponseWriter, req *http.Request) bool {
	if r.cfg.UIPassword == "" {
		return true
	}
	cookie, err := req.Cookie("tossit-ui-auth")
	if err == nil && verifyCookie(cookie.Value) {
		return true
	}
	r.serveLoginPage(w, "", false)
	return false
}

func (r *Relay) checkAdminPassword(w http.ResponseWriter, req *http.Request) bool {
	if r.cfg.AdminPassword == "off" || r.cfg.AdminPassword == "" {
		return true
	}
	cookie, err := req.Cookie("tossit-admin-auth")
	if err == nil && verifyCookie(cookie.Value) {
		return true
	}
	http.Error(w, "unauthorized", http.StatusUnauthorized)
	return false
}

func (r *Relay) HandleLogin(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {
		r.serveLoginPage(w, "", false)
		return
	}
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	password := req.FormValue("password")
	if password != r.cfg.UIPassword {
		r.serveLoginPage(w, "Invalid password", false)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "tossit-ui-auth",
		Value:    signCookie("ui-" + fmt.Sprintf("%d", time.Now().Unix())),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})
	http.Redirect(w, req, "/", http.StatusSeeOther)
}

func (r *Relay) HandleAdminLogin(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {
		r.serveLoginPage(w, "", true)
		return
	}
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	password := req.FormValue("password")
	if password != r.cfg.AdminPassword {
		r.serveLoginPage(w, "Invalid password", true)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "tossit-admin-auth",
		Value:    signCookie("admin-" + fmt.Sprintf("%d", time.Now().Unix())),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})
	if req.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
		return
	}
	http.Redirect(w, req, "/#/admin", http.StatusSeeOther)
}

func (r *Relay) serveLoginPage(w http.ResponseWriter, errMsg string, admin bool) {
	title := "tossit relay"
	action := "/api/login"
	if admin {
		title = "tossit admin"
		action = "/api/login/admin"
	}
	errorHTML := ""
	if errMsg != "" {
		errorHTML = `<p style="color:#ef4444;font-size:13px;margin-bottom:16px">` + errMsg + `</p>`
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<link href="https://fonts.googleapis.com/css2?family=DM+Sans:wght@400;500;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{background:#050505;color:#d4d4d8;font-family:"DM Sans",sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;padding:20px}
.card{background:#0a0a0c;border:1px solid #1a1a1f;border-radius:8px;padding:40px;max-width:380px;width:100%%}
.logo{font-family:"JetBrains Mono",monospace;font-size:14px;font-weight:700;color:#A3E635;margin-bottom:24px}
.logo span{color:#52525b}
input{width:100%%;background:#050505;border:1px solid #1a1a1f;border-radius:4px;padding:10px 14px;color:#d4d4d8;font-family:"JetBrains Mono",monospace;font-size:13px;margin-bottom:16px;outline:none;transition:border-color .2s}
input:focus{border-color:rgba(163,230,53,.4)}
button{width:100%%;background:#A3E635;color:#050505;border:none;border-radius:4px;padding:10px;font-family:"JetBrains Mono",monospace;font-size:13px;font-weight:700;cursor:pointer;transition:background .2s}
button:hover{background:#65A30D}
</style>
</head>
<body>
<div class="card">
<div class="logo">%s<span>_</span></div>
%s
<form method="POST" action="%s">
<input type="password" name="password" placeholder="password" autofocus>
<button type="submit">Enter</button>
</form>
</div>
</body>
</html>`, title, title, errorHTML, action)
}
