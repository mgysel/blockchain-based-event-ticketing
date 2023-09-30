class API {
  constructor() {
    this.url = "http://127.0.0.1:2100";
  }

  // GET fetch
  getPath(path) {
    const options = {
      method: "GET",
      headers: {
        "x-access-token": localStorage.getItem("token"),
      },
    };
    return fetch(`${this.url}/${path}`, options).then((res) => {
      if (!res.ok) {
        throw res;
      }
      return res.json();
    });
  }

  // PUT fetch
  putPath(path, payload) {
    const options = {
      method: "PUT",
      headers: {
        "x-access-token": localStorage.getItem("token"),
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    };
    return fetch(`${this.url}/${path}`, options).then((res) => {
      if (!res.ok) {
        throw res;
      }
      return res.json();
    });
  }

  // DELETE fetch but with POST
  deletePath(path) {
    const options = {
      method: "POST",
      headers: {
        "x-access-token": localStorage.getItem("token"),
      },
    };
    return fetch(`${this.url}/${path}`, options).then((res) => {
      if (!res.ok) {
        throw res;
      }
      return res.json();
    });
  }

  // POST fetch with token
  postAuthPath(path, payload) {
    const options = {
      method: "POST",
      headers: {
        "x-access-token": localStorage.getItem("token"),
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    };
    return fetch(`${this.url}/${path}`, options).then((res) => {
      if (!res.ok) {
        throw res;
      }
      return res.json();
    });
  }

  // POST fetch without token
  postPath(path, payload) {
    const options = {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    };
    return fetch(`${this.url}/${path}`, options).then((res) => {
      if (!res.ok) {
        throw res;
      }
      return res.json();
    });
  }
}

export default new API();
