fetchFailed = false
fetchIntervalId = -1

const BACKEND_URL = `http://${location.hostname}:${BACKEND_PORT}`

function StartFetcher(intervalTime) {
    fetchFailed = false
    fetchIntervalId = setInterval(function() {
        if (fetchFailed) {
            // if failed, stop fetching: something is wrong with server
            clearInterval(fetchIntervalId)

            console.log("Fetching data failed, stopping fetch interval.")

            return
        }
        if (document.hidden) {
            // dont bother if hidden
            return
        }
        FetchData(BACKEND_URL)
    }, intervalTime);
}

function CallFetcher() {
    if (fetchIntervalId != -1) {
        clearInterval(fetchIntervalId)
    }

    FetchData(BACKEND_URL)

    StartFetcher(5000)
}

CallFetcher()

function onFetchFail(reason) {
    console.log(reason)
    // already got err, dont change it
    if (fetchFailed == true) {
        return
    }

    clearInterval(fetchIntervalId)
    fetchFailed = true

    alert(`Fetching update failed: ${reason}\n\nFetching stopped.`)
}

function FetchData(baseUrl) {    
    try {
        var xmlHttp = new XMLHttpRequest();
        xmlHttp.onerror = function() {
            onFetchFail("Request error.")
        }
        xmlHttp.onreadystatechange = function() { 
            if (xmlHttp.readyState == 4 && xmlHttp.status == 200)
            onFetchComplete(xmlHttp.responseText);
        }
        xmlHttp.open( "GET", baseUrl+"/backends", true );
        xmlHttp.send( null );
    } catch {
        onFetchFail("Error while making request.")
    }
}

function onFetchComplete(resp) {
    data = null
    
    if (resp) {
        try {
            data = JSON.parse(resp)
        } catch {
            data = null
            onFetchFail("Invalid json data.")
        }
    }

    constructTable(data)
}

function constructTable(backendData) {
    content = ""
    if (backendData != null) {
        try {
            backendData.forEach(e => {
                content += `
                    <tr>
                        <td>${e.host}</td>
                        <td>${e.port}</td>
                        <td class="${e.alive ? "alive" : "dead"}">${e.alive ? "ALIVE" : "DEAD"}</td>
                        <td class="noborder"><button onclick="DeleteBackend('${e.host}',${e.port})">Delete</button></td>
                    </tr>
                `
            });
        }
        catch {
            content = ""
            onFetchFail("Invalid backend data.")
        }
    }

    tableElem = document.querySelector("#backend-table-data")

    tableElem.innerHTML = content
}