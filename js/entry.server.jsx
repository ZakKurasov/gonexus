import React from "react"
import {renderToString} from "react-dom/server"
import {NexusServer} from "./nexus";

export default function handleRequest(ctx) {
    const markup = renderToString(<NexusServer ctx={ctx} />)
    return "<!DOCTYPE html>" + markup;
}