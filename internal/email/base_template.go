package email

import (
	"bytes"
	"html/template"
)

// BaseEmailData contains data for the base email wrapper
type BaseEmailData struct {
	Content template.HTML
	Subject string
}

// baseEmailTemplate is the reusable wrapper for all emails
const baseEmailTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Subject}}</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            margin: 0;
            padding: 0;
            background-color: #f5f5f5;
        }
        .email-wrapper {
            max-width: 600px;
            margin: 0 auto;
            background-color: #ffffff;
        }
        .header {
            background-color: #E85D5D;
            padding: 20px 30px;
            display: flex;
            align-items: center;
            justify-content: space-between;
        }
        .header-left {
            display: flex;
            align-items: center;
            gap: 15px;
        }
        .logo-circle {
            width: 50px;
            height: 50px;
            background-color: #3D3D3D;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            flex-shrink: 0;
        }
        .logo-circle img {
            width: 35px;
            height: auto;
            filter: brightness(0) invert(1);
        }
        .brand-info {
            display: flex;
            flex-direction: column;
            gap: 2px;
        }
        .brand-name {
            font-size: 20px;
            font-weight: 700;
            color: #ffffff;
            margin: 0;
            line-height: 1;
        }
        .brand-tagline {
            font-size: 12px;
            color: rgba(255, 255, 255, 0.9);
            margin: 0;
        }
        .header-right {
            text-align: right;
        }
        .website-link {
            color: #ffffff;
            text-decoration: none;
            font-weight: 600;
            font-size: 14px;
            background-color: #3D3D3D;
            padding: 8px 16px;
            border-radius: 4px;
            display: inline-block;
        }
        .website-link:hover {
            background-color: #2a2a2a;
        }
        .content {
            padding: 30px 20px;
        }
        .footer {
            background-color: #3D3D3D;
            color: #cccccc;
            padding: 30px 20px;
            text-align: center;
            font-size: 13px;
            position: relative;
        }
        .footer-dinos {
            position: absolute;
            bottom: 0;
            left: 0;
            right: 0;
            display: flex;
            justify-content: space-between;
            align-items: flex-end;
            padding: 0 30px;
            pointer-events: none;
            opacity: 0.3;
        }
        .footer-dinos img {
            height: 60px;
            width: auto;
        }
        .footer-content {
            position: relative;
            z-index: 1;
        }
        .footer a {
            color: #E85D5D;
            text-decoration: none;
        }
        .footer a:hover {
            text-decoration: underline;
        }
        .footer-divider {
            height: 1px;
            background-color: #555;
            margin: 15px 0;
        }
        @media only screen and (max-width: 600px) {
            .header {
                flex-direction: column;
                align-items: flex-start;
                gap: 15px;
                padding: 15px 20px;
            }
            .header-left {
                width: 100%;
            }
            .header-right {
                width: 100%;
                text-align: left;
            }
            .brand-name {
                font-size: 18px;
            }
            .content {
                padding: 20px 15px;
            }
        }
    </style>
</head>
<body>
    <div class="email-wrapper">
        <!-- Modern Horizontal Header -->
        <div class="header">
            <div class="header-left">
                <div class="logo-circle">
                    <img src="https://www.logans3dcreations.com/public/images/favicon.png" alt="Logan's 3D Creations Logo" />
                </div>
                <div class="brand-info">
                    <div class="brand-name">Logan's 3D Creations</div>
                    <div class="brand-tagline">Quality 3D Prints & Designs</div>
                </div>
            </div>
            <div class="header-right">
                <a href="https://www.logans3dcreations.com" class="website-link">Visit Store →</a>
            </div>
        </div>

        <!-- Email Content -->
        <div class="content">
            {{.Content}}
        </div>

        <!-- Footer -->
        <div class="footer">
            <!-- Subtle Flexi Dinos as Background -->
            <div class="footer-dinos">
                <img src="https://www.logans3dcreations.com/public/images/flexi-trex.png" alt="" style="transform: scaleX(-1);" />
                <img src="https://www.logans3dcreations.com/public/images/flexi-trex.png" alt="" />
            </div>

            <!-- Footer Content -->
            <div class="footer-content">
                <strong style="color: #fff; font-size: 15px;">Logan's 3D Creations</strong>
                <div class="footer-divider"></div>
                <a href="mailto:prints@logans3dcreations.com">prints@logans3dcreations.com</a>
                <span style="color: #666; margin: 0 8px;">•</span>
                <a href="https://www.logans3dcreations.com">www.logans3dcreations.com</a>
                <div style="margin-top: 20px; font-size: 11px; color: #888;">
                    © 2025 Logan's 3D Creations. All rights reserved.
                </div>
            </div>
        </div>
    </div>
</body>
</html>
`

// WrapEmailContent wraps content in the base email template
func WrapEmailContent(content string, subject string) (string, error) {
	tmpl := template.Must(template.New("base").Parse(baseEmailTemplate))

	data := BaseEmailData{
		Content: template.HTML(content),
		Subject: subject,
	}

	var result bytes.Buffer
	if err := tmpl.Execute(&result, data); err != nil {
		return "", err
	}

	return result.String(), nil
}
