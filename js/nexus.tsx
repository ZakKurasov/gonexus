import React from "react"

interface NexusRoutePageModule {
    default: React.FC<any>
}

interface NexusRoute {
    path: string,
    view: NexusRoutePageModule
}

interface NexusContext {
    routes: NexusRoute[],
    props: any
}

interface NexusServerProps {
    url: string,
    context: NexusContext
}

function throwIfNull<T>(value: T | null, msg?: string): T {
    if (value === null) {
        throw new Error(msg ?? "value is null")
    }
    return value
}

const UrlContext = React.createContext<string|null>(null)
const UrlProvider: React.FC<React.PropsWithChildren<{url: string}>> = (props) => {
    const {url, children} = props
    return <UrlContext.Provider value={url}>{children}</UrlContext.Provider>
}
const useUrl = () => throwIfNull(React.useContext(UrlContext))

const NexusContextContext = React.createContext<NexusContext|null>(null)
const NexusContextProvider: React.FC<React.PropsWithChildren<{context: NexusContext}>> = (props) => {
    const {context, children} = props
    return <NexusContextContext.Provider value={context}>{children}</NexusContextContext.Provider>
}
const useNexusContext = () => throwIfNull(React.useContext(NexusContextContext))

const NexusRouter: React.FC = () => {
    const {routes, props} = useNexusContext()
    const url = useUrl()
    const route = routes.find(route => route.path === url)
    if (!route) {
        return <h1>Not found :c</h1>
    } 
    const RouteView = route.view.default
    return <RouteView {...props} />
}

export const NexusServer: React.FC<NexusServerProps> = (props) => {
    const {context, url} = props
    const route = context.routes.find(route => route.path === url)
    if (!route) {
        return <h1>404</h1>
    }
    return (
        <NexusContextProvider context={context}>
            <UrlProvider url={url}>
                <NexusRouter />
            </UrlProvider>
        </NexusContextProvider>
    )
}
