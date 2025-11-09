package email

import (
	"bytes"
	"html/template"
)

// BaseEmailData contains data for the base email wrapper
type BaseEmailData struct {
	Content          template.HTML
	Subject          string
	UnsubscribeToken string // Optional - for marketing emails only
}

// baseEmailTemplate is the reusable wrapper for all emails
// Uses table-based layout with inline styles for email client compatibility
const baseEmailTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Subject}}</title>
</head>
<body style="margin: 0; padding: 0; background-color: #f5f5f5; font-family: Arial, Helvetica, sans-serif; font-size: 14px; color: #333; line-height: 1.6;">
    <!-- Outer wrapper table for background -->
    <table width="100%" cellpadding="0" cellspacing="0" border="0" style="width: 100%; background-color: #f5f5f5;">
        <tr>
            <td align="center" style="padding: 20px 0;">
                <!-- Main email wrapper - 600px max width -->
                <table width="600" cellpadding="0" cellspacing="0" border="0" style="width: 600px; background-color: #ffffff;">
                    <!-- Header -->
                    <tr>
                        <td bgcolor="#2d2d2d" style="background-color: #2d2d2d; padding: 20px 30px;">
                            <table width="100%" cellpadding="0" cellspacing="0" border="0" style="width: 100%;">
                                <tr>
                                    <!-- Logo and Brand Info -->
                                    <td align="left" style="vertical-align: middle;">
                                        <table cellpadding="0" cellspacing="0" border="0">
                                            <tr>
                                                <td style="padding-right: 15px; vertical-align: middle;">
                                                    <img src="https://www.logans3dcreations.com/public/images/favicon.png" alt="Logan's 3D Creations Logo" width="40" height="40" style="display: block; width: 40px; height: auto;" />
                                                </td>
                                                <td style="padding-left: 0;">
                                                    <div style="font-size: 20px; font-weight: bold; color: #ffffff; margin: 0; line-height: 1.1;">Logan's 3D Creations</div>
                                                    <div style="font-size: 12px; color: rgba(255, 255, 255, 0.9); margin: 2px 0 0 0;">Quality 3D Prints & Designs</div>
                                                </td>
                                            </tr>
                                        </table>
                                    </td>
                                    <!-- Visit Store Button -->
                                    <td align="right" style="vertical-align: middle; padding-left: 20px;">
                                        <table cellpadding="0" cellspacing="0" border="0">
                                            <tr>
                                                <td bgcolor="#E85D5D" style="background-color: #E85D5D; padding: 8px 16px; border-radius: 4px;">
                                                    <a href="https://www.logans3dcreations.com" style="color: #ffffff; text-decoration: none; font-weight: 600; font-size: 14px; display: block; white-space: nowrap;">Visit Store →</a>
                                                </td>
                                            </tr>
                                        </table>
                                    </td>
                                </tr>
                            </table>
                        </td>
                    </tr>

                    <!-- Content -->
                    <tr>
                        <td style="padding: 30px 20px;">
                            {{.Content}}
                        </td>
                    </tr>

                    <!-- Footer -->
                    <tr>
                        <td bgcolor="#3D3D3D" style="background-color: #3D3D3D; color: #cccccc; padding: 30px 20px; text-align: center; font-size: 13px;">
                            <strong style="color: #ffffff; font-size: 15px; display: block; margin-bottom: 15px;">Logan's 3D Creations</strong>
                            <div style="height: 1px; background-color: #555; margin: 15px 0;"></div>
                            <div style="margin-bottom: 10px;">
                                <a href="mailto:prints@logans3dcreations.com" style="color: #E85D5D; text-decoration: none;">prints@logans3dcreations.com</a>
                                <span style="color: #666; margin: 0 8px;">•</span>
                                <a href="https://www.logans3dcreations.com" style="color: #E85D5D; text-decoration: none;">www.logans3dcreations.com</a>
                            </div>
                            <div style="font-size: 11px; color: #999; margin-bottom: 15px;">
                                25580 County Highway S, Cadott WI 54727
                            </div>
                            {{if .UnsubscribeToken}}
                            <div style="margin-top: 15px; padding-top: 15px; border-top: 1px solid #555; font-size: 11px;">
                                <a href="https://www.logans3dcreations.com/unsubscribe/{{.UnsubscribeToken}}" style="color: #999; text-decoration: none;">
                                    Unsubscribe from marketing emails
                                </a>
                            </div>
                            {{end}}
                            <div style="margin-top: 20px; font-size: 11px; color: #999;">
                                © 2025 Logan's 3D Creations. All rights reserved.
                            </div>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>
`

// WrapEmailContent wraps content in the base email template
func WrapEmailContent(content string, subject string) (string, error) {
	return WrapEmailContentWithUnsubscribe(content, subject, "")
}

// WrapEmailContentWithUnsubscribe wraps content in the base email template with optional unsubscribe link
func WrapEmailContentWithUnsubscribe(content string, subject string, unsubscribeToken string) (string, error) {
	tmpl := template.Must(template.New("base").Parse(baseEmailTemplate))

	data := BaseEmailData{
		Content:          template.HTML(content),
		Subject:          subject,
		UnsubscribeToken: unsubscribeToken,
	}

	var result bytes.Buffer
	if err := tmpl.Execute(&result, data); err != nil {
		return "", err
	}

	return result.String(), nil
}
