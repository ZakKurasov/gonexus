import React from "react"

export default function App({ children }) {
    return <html>
        <head>
            <title>Hello, world</title>
        </head>
        <body>
            {children}
        </body>
    </html>
}