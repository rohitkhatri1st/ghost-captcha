// Package ghostcaptcha renders text as an animated "ghost text" CAPTCHA: an
// image where the letterforms are invisible in any single frame and only
// become readable once the animation plays.
//
// Every pixel — inside the letterforms and outside them — is filled with
// noise drawn from the same color distribution and in constant motion, so a
// single frame is uniform static with no static patch of pixels for a
// frame-differencing pass to key on. The noise inside the letterforms
// scrolls one direction; the noise around them scrolls the opposite
// direction. That contrast is invisible in a still frame but reads
// instantly to a human eye once the animation plays, and the animation is
// exactly one seamless loop (the last frame flows straight back into the
// first).
//
// # Why
//
// An ordinary CAPTCHA is a single static image, so a bot — or an AI model
// with vision — sees exactly what a human sees and can attack it in one
// shot: OCR, frame analysis, a single call to a vision model. Because
// ghost-captcha's letterforms don't exist in any one frame, an automated
// solver has to reconstruct motion across dozens of frames instead of
// reading a single image, while a human just watches it play.
//
// This is not a guarantee against every automated solver, now or ever — no
// CAPTCHA scheme is. As of this writing, it takes considerably more effort
// for even a capable AI model to defeat than a static text CAPTCHA, and
// simpler solvers (plain OCR, single-frame vision models) can't solve it
// at all. Treat it as raising the cost of automation, not as an absolute
// guarantee, and pair it with other abuse-prevention measures (rate
// limiting, behavioral signals) for anything security-critical.
//
// # Quick start
//
// [GenerateCaptcha] is the batteries-included entry point: it draws random
// text from a charset, renders it, and returns both the text (so the caller
// can check a user's answer) and the encoded image:
//
//	text, data, err := ghostcaptcha.GenerateCaptcha(ghostcaptcha.CaptchaOptions{})
//
// [GenerateGhost] renders arbitrary caller-supplied text instead of random
// captcha text, for any other "text that resists automated reading" use case:
//
//	data, err := ghostcaptcha.GenerateGhost("HELLO", &ghostcaptcha.GhostOptions{})
//
// Both return encoded image bytes: an animated GIF by default, or WebM/MP4
// video if [GhostOptions].Format is set to [FormatWebM] or [FormatMP4]
// (which need ffmpeg on PATH; GIF output does not). Call
// [GenerateGhostFrames] directly instead of [GenerateGhost] to get the
// rendered frames themselves and encode them yourself — into GIF/WebM/MP4
// with different settings, or into a format this package doesn't support at
// all. [GhostOptions].Encoder offers a middle ground: reuse
// [GenerateGhost]'s rendering and Format/ffmpeg handling, but supply your
// own encoding step in place of its defaults.
//
// # Customization
//
// [GhostOptions] controls every visual and timing detail: font, size,
// letter spacing, canvas size, noise colors, cell size, frame count/delay,
// and how the letterforms wander across frames ([TextDrift]). Left unset,
// Width and Height default to fit the rendered text itself (longer text, or
// text with more "\n"-separated lines, defaults to a larger canvas), so most
// callers only need to set the fields they actually care about.
package ghostcaptcha
