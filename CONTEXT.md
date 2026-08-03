# Domain Glossary — go-http-server

## Chirp

A short text post (max 140 characters, profanity-filtered) authored by a User.

## User

An account identified by email, authenticated with a password (argon2id-hashed). A User owns the Chirps it creates.

## Session

The thing a login produces: a short-lived access JWT plus a long-lived refresh token. Revocation ends a Session.

## Authed handler

A handler that receives a validated `userID` as a parameter instead of spelunking the request context. The auth handshake (bearer parse, JWT validation, 401 flow) lives entirely behind the `RequireAuth` seam, so an authed handler can never see a missing or forged userID.

## Auth handshake

The transformation from "an HTTP request with an `Authorization: Bearer` header" to "a validated `userID`". Lives behind `RequireAuth` in `internal/handler/middleware.go`.
