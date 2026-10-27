// React wrapper around the `reicon` icon library. reicon icon functions expose a
// `.toSvg(options)` method that returns an SVG string — perfect for React via
// dangerouslySetInnerHTML. Missing icons fall back gracefully.
import {
  Home, Users, User, Pill, Stethoscope, Calendar, Calendar2, Settings, Plus, Edit,
  Trash, Search, ChevronDown, Language, Logout, Menu, Check, AlertTriangle, Flask,
  Heart, Activity, Clipboard, Note, Notebook, Dollar, Dollar2, Globe, Refresh, Print,
  Filter, Box, Box2, Hospital, Phone, Tag, Save, Eye, ArrowRight, ArrowLeft, X, Grid,
  ChevronRight, Lock, ChevronLeft,
} from 'reicon';

const ICONS = {
  Home, Users, User, Pill, Stethoscope, Calendar, Calendar2, Settings, Plus, Edit,
  Trash, Search, ChevronDown, Language, Logout, Menu, Check, AlertTriangle, Flask,
  Heart, Activity, Clipboard, Note, Notebook, Dollar, Dollar2, Globe, Refresh, Print,
  Filter, Box, Box2, Hospital, Phone, Tag, Save, Eye, ArrowRight, ArrowLeft, X, Grid,
  ChevronRight, Lock, ChevronLeft,
};

// Map semantic names that may not exist in the set to verified alternatives.
const FALLBACK = {
  dashboard: 'Home',
  usergroup: 'Users',
  close: 'X',
  medical: 'Pill',
  firstaid: 'Heart',
  cross: 'Plus',
  mail: 'User',
  gender: 'User',
  grid: 'Grid',
  info: 'AlertTriangle',
  checkcircle: 'Check',
};

export default function Reicon({ name, size = 20, color = 'currentColor', weight = 'Outline', className = '', style = {} }) {
  const key = FALLBACK[name] || name;
  const Cmp = ICONS[key];
  if (!Cmp || typeof Cmp.toSvg !== 'function') {
    return (
      <span
        className={`reicon-wrapper ${className}`}
        style={{ width: size, height: size, display: 'inline-flex', ...style }}
      />
    );
  }
  const svg = Cmp.toSvg({ size, color, weight });
  return (
    <span
      className={`reicon-wrapper ${className}`}
      style={{ width: size, height: size, display: 'inline-flex', lineHeight: 0, ...style }}
      dangerouslySetInnerHTML={{ __html: svg }}
    />
  );
}
