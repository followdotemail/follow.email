# How to Parse Gmail API Email Messages

Based on the official Gmail API documentation, here's the proper way to parse emails retrieved from the Gmail API.

## Official Approach

When you call the Gmail API's `users.messages.get` method, you need to use the `format` parameter to control how the message is returned. The recommended format for parsing email content is **`full`**, which returns the parsed email structure in the `payload` field.

## Step-by-Step Parsing Process

### 1. Fetch the Message with Full Format

Use the `users.messages.get` endpoint with `format=full`:

```python
message = service.users().messages().get(userId='me', id=message_id, format='full').execute()
```

### 2. Understanding the Message Structure

The response contains a `payload` object representing the MIME structure of the email. The key components are:

- **`payload.headers`**: Contains email headers (Subject, From, To, Date, etc.)
- **`payload.body`**: Contains the message body data (for single-part messages)
- **`payload.parts`**: Contains multiple parts for multipart MIME messages (plain text, HTML, attachments)
- **`payload.mimeType`**: Indicates the content type (e.g., `text/plain`, `text/html`, `multipart/alternative`)

### 3. Extract Email Body Content

The message body is **base64url encoded** and must be decoded. Here's the official parsing logic:

#### For Single-Part Messages:

```python
import base64

if 'body' in payload and 'data' in payload['body']:
    data = payload['body']['data']
    decoded_bytes = base64.urlsafe_b64decode(data.encode('UTF-8'))
    body_text = decoded_bytes.decode('UTF-8')
```

#### For Multipart Messages (Most Common):

```python
def get_message_body(payload):
    if 'body' in payload and 'data' in payload['body']:
        data = payload['body']['data']
        decoded_bytes = base64.urlsafe_b64decode(data.encode('UTF-8'))
        return decoded_bytes.decode('UTF-8')
    elif 'parts' in payload:
        for part in payload['parts']:
            # For plain text
            if part['mimeType'] == 'text/plain':
                data = part['body']['data']
                decoded_bytes = base64.urlsafe_b64decode(data.encode('UTF-8'))
                return decoded_bytes.decode('UTF-8')
            # For HTML
            elif part['mimeType'] == 'text/html':
                data = part['body']['data']
                decoded_bytes = base64.urlsafe_b64decode(data.encode('UTF-8'))
                return decoded_bytes.decode('UTF-8')
            # For nested parts (recursive)
            elif 'parts' in part:
                return get_message_body(part)
    return None
```

### 4. Handle Multipart/Alternative Messages

Many emails come in `multipart/alternative` format with both plain text and HTML versions. You should check for both:

- **`text/plain`**: Plain text version
- **`text/html`**: HTML version with formatting

Choose which version to use based on your requirements. If you need to preserve formatting, extract the HTML version; for simple text processing, use plain text.

### 5. Important Considerations

- **Base64url encoding**: Gmail uses base64url encoding (not standard base64), so always use `base64.urlsafe_b64decode`
- **Recursive parsing**: Email structures can be deeply nested, so implement recursive parsing for `parts`
- **MIME type checking**: Always check the `mimeType` to handle different content types correctly
- **Attachments**: If a part has an `attachmentId` in the body, it's an attachment that requires a separate API call to `messages.attachments.get`

## Official Documentation Links

Here are the official Google documentation links for the Gmail API message parsing:

1. **Gmail API Reference - users.messages**: https://developers.google.com/gmail/api/reference/rest/v1/users.messages
2. **Format Parameter Documentation**: https://developers.google.com/gmail/api/reference/rest/v1/Format
3. **Message Structure (MessagePart)**: https://developers.google.com/gmail/api/reference/rest/v1/users.messages#MessagePart
4. **Sending Email Guide** (shows MIME structure): https://developers.google.com/gmail/api/guides/sending
5. **List Gmail Messages Guide**: https://developers.google.com/gmail/api/guides/list

## Complete Working Example (Python)

```python
import base64
from googleapiclient.discovery import build

def parse_message(service, message_id):
    # Get message with full format
    message = service.users().messages().get(
        userId='me', 
        id=message_id, 
        format='full'
    ).execute()
    
    # Extract headers
    headers = message['payload']['headers']
    subject = next(h['value'] for h in headers if h['name'] == 'Subject')
    from_email = next(h['value'] for h in headers if h['name'] == 'From')
    
    # Extract body
    payload = message['payload']
    body = extract_body(payload)
    
    return {
        'subject': subject,
        'from': from_email,
        'body': body
    }

def extract_body(payload):
    """Recursively extract email body from payload"""
    if 'body' in payload and 'data' in payload['body']:
        return base64.urlsafe_b64decode(
            payload['body']['data']
        ).decode('utf-8')
    
    if 'parts' in payload:
        for part in payload['parts']:
            if part['mimeType'] == 'text/plain':
                return base64.urlsafe_b64decode(
                    part['body']['data']
                ).decode('utf-8')
        # If no plain text, try HTML
        for part in payload['parts']:
            if part['mimeType'] == 'text/html':
                return base64.urlsafe_b64decode(
                    part['body']['data']
                ).decode('utf-8')
        # Recursively check nested parts
        for part in payload['parts']:
            if 'parts' in part:
                result = extract_body(part)
                if result:
                    return result
    
    return None
```

This approach follows the official Gmail API documentation and handles the most common email formats you'll encounter when parsing Gmail messages.
