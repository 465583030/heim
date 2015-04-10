var React = require('react/addons')

var Main = require('./ui/main')

var roomName = location.pathname.match(/(\w+)\/$/)[1]
document.title = roomName
require('./actions').connect(roomName)

if (React.addons && React.addons.Perf) {
  ReactPerf = React.addons.Perf
  if (location.hash == '#perf') {
    ReactPerf.start()
  }
}

React.render(
  <Main />,
  document.getElementById('container')
)

Heim = {
  actions: require('./actions'),
  socket: require('./stores/socket'),
  chat: require('./stores/chat'),
  storage: require('./stores/storage'),
  focus: require('./stores/focus'),
}

document.body.addEventListener('keydown', function(ev) {
  if (ev.target == document.body && ev.which == 8) {
    ev.preventDefault()
  }
}, false)
