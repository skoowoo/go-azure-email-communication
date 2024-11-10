# azure-email-communication

A Go package for sending emails using Azure Communication Services.

## Installation

```bash
go get github.com/skoowoo/go-azure-email-communication
```

## Quick Start

```go
package main

import (
    "log"
    azmail "github.com/skoowoo/go-azure-email-communication"
)

func main() {
    // Initialize the client
    client, err := azmail.NewClient(
        azmail.WithMailFrom("sender@yourdomain.com"),
        azmail.WithEndpoint("https://your-resource.unitedstates.communication.azure.com", "your-access-key"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Send an email
    err = client.SendMail(
        "recipient@example.com",
        "Hello from Azure",
        "<h1>Hello!</h1><p>This is a test email.</p>",
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

## License

MIT
