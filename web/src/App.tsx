import { Outlet } from 'react-router-dom'

function App() {
  return (
    <div className="flex flex-col w-screen h-svh max-h-svh">
      <div className="flex w-full">
        <div className="sticky top-0 w-full">
          <div className="p-5">
            <h1 className="text-left text-xl">Header</h1>
          </div>
       </div>
      </div>
      <div className="flex flex-row w-full h-full flex-1">
        <div className="h-full w-1/5 p-4">
          <h1 className="text-xl">Left</h1>
        </div>
        <div className="h-full flex-1 p-4">
          <Outlet />
        </div>
      </div>
    </div>
  )
}

export default App
