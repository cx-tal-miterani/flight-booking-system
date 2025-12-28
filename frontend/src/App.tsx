import { Routes, Route } from 'react-router-dom';
import { FlightList } from './components/FlightList';
import { BookingPage } from './components/BookingPage';

function App() {
  return (
    <div className="min-h-screen">
      <header className="border-b border-slate-700/50 backdrop-blur-sm sticky top-0 z-50 bg-slate-900/80">
        <div className="max-w-7xl mx-auto px-6 py-4">
          <a href="/" className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-cyan-500 to-emerald-500 flex items-center justify-center">
              <svg className="w-6 h-6 text-slate-900" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
              </svg>
            </div>
            <span className="font-semibold text-2xl text-white">SkyBook</span>
          </a>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-6 py-8">
        <Routes>
          <Route path="/" element={<FlightList />} />
          <Route path="/book/:flightId" element={<BookingPage />} />
        </Routes>
      </main>

      <footer className="border-t border-slate-700/50 mt-auto">
        <div className="max-w-7xl mx-auto px-6 py-6 text-center text-slate-500 text-sm">
          <p>Flight Booking System - Powered by Temporal</p>
        </div>
      </footer>
    </div>
  );
}

export default App;

