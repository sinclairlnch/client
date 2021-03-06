import fs from 'fs'
import os from 'os'
import path from 'path'
import {WriteStream, Encoding} from './file'

export const downloadFolder = __STORYBOOK__
  ? ''
  : process.env.XDG_DOWNLOAD_DIR || path.join(os.homedir(), 'Downloads')

export function writeStream(filepath: string, encoding: string, append?: boolean): Promise<WriteStream> {
  const ws = fs.createWriteStream(filepath, {encoding, flags: append ? 'a' : 'w'})
  return Promise.resolve({
    close: () => ws.end(),
    write: d => {
      ws.write(d)
      return Promise.resolve()
    },
  })
}

export function readFile(filepath: string, encoding: Encoding): Promise<any> {
  return new Promise((resolve, reject) => {
    fs.readFile(filepath, {encoding}, (err, data) => {
      if (err) {
        reject(err)
      } else {
        resolve(data)
      }
    })
  })
}
