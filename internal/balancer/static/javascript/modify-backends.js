function DeleteBackend(host, port) {
    backend = {
        host:host, port:port
    }

    payload = JSON.stringify(backend)

    try {
        var xmlHttp = new XMLHttpRequest();
        xmlHttp.onerror = function() {
            console.log("Delete backend error.")
        }
        xmlHttp.onreadystatechange = function() { 
            if (xmlHttp.readyState == 4 && xmlHttp.status == 200) {
                console.log("done")
                CallFetcher()
            }
        }
        xmlHttp.open( "DELETE", BACKEND_URL+"/backends", true );
        xmlHttp.send( payload );
    } catch (e) {
        console.log(e)
    }
}

function AddBackend(host,port) {
    backend = {
        host:host, port:port
    }

    payload = JSON.stringify(backend)

    try {
        var xmlHttp = new XMLHttpRequest();
        xmlHttp.onerror = function() {
            console.log("Create backend error.")
        }
        xmlHttp.onreadystatechange = function() { 
            if (xmlHttp.readyState == 4 && xmlHttp.status == 200) {
                console.log("done")
                CallFetcher()
            }
        }
        xmlHttp.open( "PUT", BACKEND_URL+"/backends", true );
        xmlHttp.send( payload );
    } catch (e) {
        console.log(e)
    }
}

const addBackendFormElem = document.getElementById("addbackendform")

function handleAddBackendForm(event) {
    event.preventDefault()

    const formData = new FormData(addBackendFormElem)

    host = formData.get("host")
    port = parseInt(formData.get("port"))

    AddBackend(host,port)

    formData.set("host", "")
    formData.set("port", "")
}  

addBackendFormElem.addEventListener('submit', handleAddBackendForm)