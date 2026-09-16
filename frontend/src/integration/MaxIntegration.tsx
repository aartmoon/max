import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { useLocation, useNavigate } from "react-router-dom";
import {
  browserBridge,
  loadMaxBridge,
  launchRoute,
  type MaxBridge,
} from "./max";
const BridgeContext = createContext<MaxBridge>(browserBridge);
export const useMaxBridge = () => useContext(BridgeContext);
function parentRoute(path: string): string {
  if (path.startsWith("/admin/requests/")) return "/admin";
  if (path.startsWith("/requests/") && path !== "/requests/new")
    return "/requests";
  return "/";
}
export function MaxIntegration({ children }: { children: ReactNode }) {
  const [bridge, setBridge] = useState<MaxBridge>(browserBridge);
  const location = useLocation(),
    navigate = useNavigate();
  const launchHandled = useRef(false);
  const initialLocation = useRef(location.key);
  useEffect(() => {
    let active = true;
    loadMaxBridge().then((value) => {
      if (active) setBridge(value);
    });
    return () => {
      active = false;
    };
  }, []);
  useEffect(() => {
    if (bridge.platform !== "max") return;
    bridge.ready();
    if (!launchHandled.current) {
      launchHandled.current = true;
      const target = launchRoute(bridge.launchData().startParam);
      // A late SDK must not interrupt a user who has already navigated.
      if (
        target &&
        location.pathname === "/" &&
        location.key === initialLocation.current
      )
        navigate(target, { replace: true });
    }
  }, [bridge, location.pathname, location.key, navigate]);
  useEffect(
    () =>
      bridge.backButton(
        location.pathname === "/"
          ? null
          : () => {
              if (
                typeof window.history.state?.idx === "number" &&
                window.history.state.idx > 0
              )
                navigate(-1);
              else navigate(parentRoute(location.pathname), { replace: true });
            },
      ),
    [bridge, location.pathname, navigate],
  );
  return (
    <BridgeContext.Provider value={bridge}>{children}</BridgeContext.Provider>
  );
}
