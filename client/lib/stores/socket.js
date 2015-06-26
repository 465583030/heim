var _ = require('lodash')
var url = require('url')
var Reflux = require('reflux')


var storeActions = Reflux.createActions([
  'send',
  'pingIfIdle',
  'connect',
])
_.extend(module.exports, storeActions)

storeActions.connect.sync = true

function logPacket(kind, data) {
  console.groupCollapsed(kind, data.type, data.id ? '(id: ' + data.id + ')' : '(no id)')
  console.log(data)
  console.log(JSON.stringify(data, true, 2))
  console.groupEnd()
}

module.exports.store = Reflux.createStore({
  listenables: storeActions,

  pingLimit: 2000,

  init: function() {
    this.ws = null
    this.seq = 0
    this.pingTimeout = null
    this.pingReplyTimeout = null
    this.nextPing = 0
    this.lastMessage = null
    this._logPackets = false
  },

  _wsurl: function(origin, prefix, roomName) {
    var parsedOrigin = url.parse(origin)

    var scheme = 'ws'
    if (parsedOrigin.protocol == 'https:') {
      scheme = 'wss'
    }

    return scheme + '://' + parsedOrigin.host + prefix + '/room/' + roomName + '/ws'
  },

  connect: function(roomName) {
    this.roomName = this.roomName || roomName
    this._connect()
  },

  _connect: function() {
    var wsurl = this._wsurl(process.env.HEIM_ORIGIN, process.env.HEIM_PREFIX, this.roomName)
    this.ws = new WebSocket(wsurl, 'heim1')
    this.ws.onopen = this._open
    this.ws.onclose = this._closeReconnectSlow
    this.ws.onmessage = this._message
  },

  _reconnect: function() {
    // forcefully drop websocket and reconnect
    this._close()
    this.ws.close()
    this._connect()
  },

  _open: function() {
    this.trigger({
      status: 'open',
    })
  },

  _close: function() {
    clearTimeout(this.pingTimeout)
    clearTimeout(this.pingReplyTimeout)
    this.pingReplyTimeout = null
    this.ws.onopen = this.ws.onclose = this.ws.onmessage = null
    this.trigger({
      status: 'close',
    })
  },

  _closeReconnectSlow: function() {
    this._close()
    var delay = 2000 + 3000 * Math.random()
    setTimeout(this._connect, delay)
  },

  _message: function(ev) {
    var data = JSON.parse(ev.data)

    if (this._logPackets) {
      logPacket('recv', data)
    }

    this.lastMessage = Date.now()

    this._handlePings(data)

    this.trigger({
      status: 'receive',
      body: data,
    })
  },

  _handlePings: function(msg) {
    if (msg.type == 'ping-event') {
      if (msg.data.next > this.nextPing) {
        var interval = msg.data.next - msg.data.time
        this.nextPing = msg.data.next
        clearTimeout(this.pingTimeout)
        this.pingTimeout = setTimeout(this._ping, interval * 1000)
      }

      this.send({
        type: 'ping-reply',
        data: {
          time: msg.data.time,
        },
      })
    }

    // receiving any message removes the need to ping
    clearTimeout(this.pingReplyTimeout)
    this.pingReplyTimeout = null
  },

  send: function(data) {
    if (!data.id) {
      data.id = String(this.seq++)
    }

    // FIXME: remove when fixed on server
    if (!data.data) {
      data.data = {}
    }

    if (this._logPackets) {
      logPacket('send', data)
    }
    this.ws.send(JSON.stringify(data))
  },

  _ping: function() {
    if (this.pingReplyTimeout) {
      return
    }

    this.send({
      type: 'ping',
    })

    this.pingReplyTimeout = setTimeout(this._reconnect, this.pingLimit)
  },

  pingIfIdle: function() {
    if (this.lastMessage === null || Date.now() - this.lastMessage >= this.pingLimit) {
      this._ping()
    }
  },
})
