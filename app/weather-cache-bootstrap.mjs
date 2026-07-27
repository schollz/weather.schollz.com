export const WEATHER_CACHE_KEY = "wx-weather-v1";
export const WEATHER_CACHE_TTL_MS = 60 * 60 * 1000;

export function weatherCacheBootstrapScript() {
  return `(function(){try{if(new URLSearchParams(location.search).has("location"))return;var p=location.pathname.replace(/^\\/+|\\/+$/g,"");p=p?"/"+p+"/":"/";var v=JSON.parse(localStorage.getItem(${JSON.stringify(WEATHER_CACHE_KEY)})||"[]");var n=Date.now();var h=Array.isArray(v)&&v.some(function(e){var a=e?n-e.updatedAt:-1;return e&&Array.isArray(e.paths)&&e.paths.indexOf(p)!==-1&&Number.isFinite(e.updatedAt)&&a>=0&&a<${WEATHER_CACHE_TTL_MS}&&e.weather&&typeof e.weather.city==="string"&&typeof e.weather.current==="object"&&Array.isArray(e.weather.hourly)&&Array.isArray(e.weather.daily)});if(h)document.documentElement.dataset.weatherCache="hit"}catch(e){}})();`;
}
