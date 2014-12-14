var React = require('react')
require('superagent-bluebird-promise')

var Main = require('./ui/main')


React.render(
  <Main />,
  document.getElementById('container')
)

require('./actions').connect()

Heim = {
  socket: require('./stores/socket'),
  chat: require('./stores/chat'),
  storage: require('./stores/storage'),
}
