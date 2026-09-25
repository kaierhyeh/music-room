package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func HomePage(c echo.Context) error {
	return c.HTML(http.StatusOK, homePage)
}

func PrivacyPage(c echo.Context) error {
	return c.HTML(http.StatusOK, privacyPage)
}

const pageHead = `<!doctype html><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<style>
 body{margin:0;min-height:100vh;display:grid;place-items:center;
      background:#0b0b0b;color:#c8c8c8;
      font:16px/1.6 ui-monospace,SFMono-Regular,Menlo,monospace}
 main{max-width:40rem;padding:2rem}
 h1{color:#4ade80;font-size:1.25rem;letter-spacing:.08em;text-transform:uppercase}
 h2{color:#4ade80;font-size:1rem;margin-top:2rem}
 a{color:#4ade80}
</style>
`

const homePage = pageHead + `<title>MusicRoom</title>
<main>
 <h1>MusicRoom</h1>
 <p>Collaborative music app: shared playlists that friends edit together, and
 live events where the crowd votes on the next track.</p>
 <p>A student project at 42. Sign in from the MusicRoom mobile app.</p>
 <p><a href="/privacy">Privacy policy</a></p>
</main>`

const privacyPage = pageHead + `<title>MusicRoom — Privacy policy</title>
<main>
 <h1>Privacy policy</h1>
 <p>MusicRoom is a student project at 42. This page lists what it stores about
 you and why.</p>

 <h2>Signing in with Google</h2>
 <p>When you sign in with Google, MusicRoom receives your Google account
 identifier, your email address and your name. The identifier ties your Google
 account to your MusicRoom account; the email and name become your account's
 email and display name. Nothing else is requested from Google, and no Google
 token is kept once sign-in completes.</p>

 <h2>What else is stored</h2>
 <ul>
  <li>Your account: email, display name, a hashed password if you set one, and
  the profile details and music preferences you choose to fill in, with the
  visibility you give each of them.</li>
  <li>What you do in the app: friends, playlists, events, the tracks you add
  and the votes you cast.</li>
  <li>For security: your active sessions and an activity log recording the IP
  address, device type and app version of each request.</li>
 </ul>
 <p>For events restricted to a place, your location is checked when you vote
 and is not stored.</p>

 <h2>Sharing</h2>
 <p>Your data is not sold or shared with third parties. Track search and audio
 previews come from Deezer: searches go through the MusicRoom server, and
 previews are fetched by the app straight from Deezer, which therefore sees your
 device's IP address as any website would.</p>

 <h2>Deletion</h2>
 <p>You can unlink Google from your profile at any time. To have your account
 and all the data above deleted, write to the support address shown on
 Google's sign-in screen.</p>
</main>`
