# TimmyGram Server

This is the backend for TimmyGramApp - an iOS application for watching short videos from the TimmyGram Server.

The server allows parents to create an Account, upload videos and connect devices with the TimmyGram App.

## Functionality

- Creates a Parent account
- Uploads videos
- Transcode videos to portrait format (16:9) and enforces length of the video to 60 seconds.
- Connects devices with the TimmyGram App, by generating a unique QR code.

## Why TimmyGram?

Allowing children to access public and open social media networks there is no filter on the content.
TimmyGram is designed to provide a safe and controlled environment for children to watch videos. 
By allowing parents to selfhost their own video sharing platform, they have full control over the content 
their children can access. This ensures that the videos are appropriate and suitable for their age group. 
The app only allows access to videos that are uploaded to the server, providing an additional layer of 
security and privacy.

## How to deploy the server?

### Docker

1. Clone the repository
2. Modify the config.yaml file to your liking
 - set db-path to `/app/data/timmygram.db`
 - set video path to `/app/data/videos`
 - update the public URL to the server in `server_url`
3. Run `docker-compose up -d` to start the server
4. Open the server on http://localhost:8080
5. Create your parent account (username and password)

### Go run

1. Clone the repository
2. Modify the config.yaml if needed
3. Run `go run main.go`
4. The rest is the same as Docker


## Demo

To check the demo, please visit: https://o3nqm8rmi03dmtmy0isvbd5t.myspidy.com
Login: `demo`
Password: `Demo873`

## TODOs

- [x] Better video feed - currently it selects a random video form the database
- [x] Likes for videos
- [x] Add a way to delete/edit videos
- [x] Add a way to disable/enable access for a device
- [ ] Eventually allow multiple parents to upload videos

## License: MIT License

Copyright 2026 Vladimir Kolev

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
