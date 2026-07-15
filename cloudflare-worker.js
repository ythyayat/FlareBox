/**
 * Cloudflare Email Worker
 * This worker receives emails and forwards them to your webhook server
 * with properly parsed content (text and HTML body)
 */

export default {
    async email(message, env, ctx) {
        // CONFIGURATION - Update these values
        const webhookUrl = "https://your-server.com/api/webhook";
        const apiKey = "your-secret-api-key-here";

        try {
            // Get the raw email as text
            const rawEmail = await streamToText(message.raw);

            // Parse the email to extract text and HTML parts
            const { textBody, htmlBody } = parseEmailContent(rawEmail);

            // Prepare the data to send
            const emailData = {
                to: message.to,
                from: message.from,
                subject: message.headers.get('subject') || 'No Subject',
                body: textBody || `Email from ${message.from}`,
                html_body: htmlBody || "",
                has_attachments: false
            };

            // Send to webhook
            const response = await fetch(webhookUrl, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-API-Key': apiKey
                },
                body: JSON.stringify(emailData)
            });

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(`Webhook failed: ${response.status} - ${errorText}`);
            }

            console.log('Email forwarded successfully');
            console.log('Subject:', emailData.subject);
            console.log('Text Body:', textBody?.substring(0, 100));

        } catch (error) {
            console.error('Error forwarding email:', error.message);
            // Don't throw - we don't want to bounce the email
        }
    }
}

/**
 * Convert ReadableStream to text
 */
async function streamToText(stream) {
    const reader = stream.getReader();
    const decoder = new TextDecoder();
    let result = '';

    while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        result += decoder.decode(value, { stream: true });
    }

    result += decoder.decode(); // flush
    return result;
}

/**
 * Parse email content to extract text and HTML bodies
 * Handles multipart/alternative emails properly
 */
function parseEmailContent(rawEmail) {
    let textBody = '';
    let htmlBody = '';

    // Find the boundary for multipart messages
    const boundaryMatch = rawEmail.match(/boundary="?([^"\s]+)"?/);

    if (boundaryMatch) {
        const boundary = boundaryMatch[1];
        const parts = rawEmail.split(`--${boundary}`);

        for (const part of parts) {
            // Check for text/plain
            if (part.includes('Content-Type: text/plain')) {
                textBody = extractBodyFromPart(part);
            }
            // Check for text/html
            if (part.includes('Content-Type: text/html')) {
                htmlBody = extractBodyFromPart(part);
            }
        }
    } else {
        // Simple email without multipart
        // Split by double newline to separate headers from body
        const parts = rawEmail.split(/\r?\n\r?\n/);
        if (parts.length > 1) {
            textBody = parts.slice(1).join('\n\n').trim();
        }
    }

    return {
        textBody: textBody.trim(),
        htmlBody: htmlBody.trim()
    };
}

/**
 * Extract body content from an email part
 */
function extractBodyFromPart(part) {
    // Split by double newline to separate headers from content
    const sections = part.split(/\r?\n\r?\n/);

    if (sections.length < 2) {
        return '';
    }

    // Get the headers section
    const headers = sections[0];

    // Join all sections after the headers and clean up
    let content = sections.slice(1).join('\n\n');

    // Remove any trailing boundary markers
    content = content.replace(/--[a-z0-9]+--\s*$/gi, '');

    // Check for Content-Transfer-Encoding
    const encodingMatch = headers.match(/Content-Transfer-Encoding:\s*([^\r\n]+)/i);
    const encoding = encodingMatch ? encodingMatch[1].trim().toLowerCase() : '7bit';

    // Decode based on encoding type
    if (encoding === 'quoted-printable') {
        content = decodeQuotedPrintable(content);
    } else if (encoding === 'base64') {
        content = decodeBase64(content);
    }

    return content.trim();
}

/**
 * Decode quoted-printable encoded text
 * Converts =XX sequences to actual characters and handles soft line breaks
 */
function decodeQuotedPrintable(text) {
    // Replace soft line breaks (= at end of line)
    text = text.replace(/=\r?\n/g, '');

    // Decode =XX sequences
    text = text.replace(/=([0-9A-F]{2})/gi, (match, hex) => {
        return String.fromCharCode(parseInt(hex, 16));
    });

    return text;
}

/**
 * Decode base64 encoded text
 */
function decodeBase64(text) {
    try {
        // Remove whitespace and newlines
        text = text.replace(/\s/g, '');
        // Decode base64
        return atob(text);
    } catch (e) {
        console.error('Failed to decode base64:', e);
        return text; // Return original if decode fails
    }
}
