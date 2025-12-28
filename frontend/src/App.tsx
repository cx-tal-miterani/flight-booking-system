import { Routes, Route } from 'react-router-dom';
import { Plane } from 'lucide-react';
import { FlightList } from './components/FlightList';
import { BookingPage } from './components/BookingPage';

function App() {
  return (
    <div className="min-h-screen flex flex-col">
      {/* Header */}
      <header className="border-b border-slate-700/50 backdrop-blur-sm sticky top-0 z-50 bg-slate-900/80">
        <div className="container-responsive py-4">
          <a href="/" className="flex items-center gap-3 w-fit">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-cyan-500 to-emerald-500 flex items-center justify-center shadow-lg shadow-cyan-500/25">
              <Plane className="w-5 h-5 text-slate-900 transform rotate-45" />
            </div>
            <div>
              <span className="font-bold text-xl text-white">SkyBook</span>
              <span className="hidden sm:inline text-slate-500 text-sm ml-2">Flight Booking</span>
            </div>
          </a>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 container-responsive py-6 sm:py-8">
        <Routes>
          <Route path="/" element={<FlightList />} />
          <Route path="/book/:flightId" element={<BookingPage />} />
        </Routes>
      </main>

      {/* Footer */}
      <footer className="border-t border-slate-700/50 mt-auto">
        <div className="container-responsive py-6 text-center">
          <p className="text-slate-500 text-sm">
            Flight Booking System — Powered by{' '}
            <a 
              href="https://temporal.io" 
              target="_blank" 
              rel="noopener noreferrer"
              className="text-cyan-500 hover:text-cyan-400 transition-colors"
            >
              Temporal
            </a>
          </p>
        </div>
      </footer>
    </div>
  );
}

export default App;
