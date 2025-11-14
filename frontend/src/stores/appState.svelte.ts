export type AlertType = "success" | "error" | "warning" | "info";
export interface DatabaseInfo {
    tables: string[];
    appCreated: boolean;
}
export interface NavDatabases {
    [key: string]: DatabaseInfo;
}
export interface QueryResults {
    pk: boolean,
    cols: string[],
    rows: string[][],
    editable: boolean
}

export interface Alert {
    msg: string,
    type: AlertType,
    show: boolean,
    duration: number
}

export interface DialogMessage {
    title: string,
    msg: string,
    options: string[],
    actions: CallableFunction[],
    btnStyles: string[],
    show: boolean
}
export interface AppState {
    rootPath: string,
    currentDB: string,
    navData: NavDatabases,
    queryResults: QueryResults,
    selectedTable: string,
    loadingQueryResults: boolean,
    alert: Alert,
    resultAlert: Alert,
    dialog: DialogMessage
}


export const appState: AppState = $state({
    rootPath: "",
    currentDB: "",
    navData: {},
    queryResults: {
        pk: false,
        cols: [],
        rows: [],
        editable: false
    },
    selectedTable: "",
    loadingQueryResults: false,
    alert: { msg: "", type: "success", show: false, duration: 3000 },
    resultAlert: { msg: "", type: "success", show: false, duration: 3000 },
    dialog: {
        title: "New Dialogue",
        msg: "",
        options: ["Cancel", "OK"],
        actions: [],
        btnStyles: [],
        show: false,
    }
})