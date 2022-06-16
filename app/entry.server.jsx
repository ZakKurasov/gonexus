import React from "react"
import {renderToStaticMarkup} from "react-dom/server"

import { NexusServer } from "../js/nexus"

export default function handleRequest(ctx, req) {
    const markup = renderToStaticMarkup(<NexusServer context={ctx} url={req.url} />)
    return new Response(`<!DOCTYPE html>`+markup, {
        status: 200,
        headers: {
            'Content-Type': 'text/html'
        }
    })
}