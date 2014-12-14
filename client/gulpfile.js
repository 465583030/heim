var _ = require('lodash')
var gulp = require('gulp')
var gutil = require('gulp-util')
var streamify = require('gulp-streamify')
var less = require('gulp-less')
var uglify = require('gulp-uglify')
var source = require('vinyl-source-stream')
var watchify = require('watchify')
var browserify = require('browserify')


var dest = './build'

function bundler(args) {
  return browserify('./lib/client.js', args)
}

gulp.task('js', function() {
  return bundler()
    .bundle()
    .pipe(source('main.js'))
    //.pipe(streamify(uglify()))
    .on('error', gutil.log.bind(gutil, 'browserify error'))
    .pipe(gulp.dest(dest))
})

gulp.task('less', function() {
  return gulp.src('./lib/main.less')
    .pipe(less({compress: true}))
    .on('error', function(err) {
      gutil.log(gutil.colors.red('LESS error:'), err.message)
    })
    .pipe(gulp.dest(dest))
})

gulp.task('html', function() {
  return gulp.src('./lib/index.html')
    .pipe(gulp.dest(dest))
})

gulp.task('watchify', function() {
  // via https://github.com/gulpjs/gulp/blob/master/docs/recipes/fast-browserify-builds-with-watchify.md
  bundler = watchify(bundler(watchify.args))
  bundler.on('log', gutil.log.bind(gutil, gutil.colors.green('Watchify')))
  bundler.on('update', rebundle)

  function rebundle() {
    return bundler.bundle()
      .on('error', function(err) {
        gutil.log(gutil.colors.red('Watchify error:'), err.message)
      })
      .pipe(source('main.js'))
      .pipe(gulp.dest(dest))
  }

  return rebundle()
})

gulp.task('watch', function () {
  gulp.watch('./lib/**/*.less', ['less'])
  gulp.watch('./lib/**/*.html', ['html'])
})

gulp.task('build', ['js', 'less', 'html'])
gulp.task('default', ['less', 'html', 'watch', 'watchify'])
