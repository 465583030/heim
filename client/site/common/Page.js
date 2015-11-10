import React from 'react'

import heimURL from '../../lib/heim-url'


export default function Page(props) {
  return (
    <html>
    <head>
      <meta charSet="utf-8" />
      <title>{props.title}</title>
      <link rel="icon" id="favicon" href={heimURL('/static/favicon.png')} sizes="32x32" />
      <link rel="icon" href={heimURL('/static/favicon-192.png')} sizes="192x192" />
      <meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no" />
      <link rel="stylesheet" type="text/css" id="css" href={heimURL('/static/site.css')} />
      <script async src={heimURL('/static/fast-touch.js')} />
    </head>
    <body className={props.className}>
      {props.children}
    </body>
    </html>
  )
}

Page.propTypes = {
  title: React.PropTypes.string,
  className: React.PropTypes.string,
  children: React.PropTypes.node,
}
