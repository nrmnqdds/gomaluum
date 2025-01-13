import { QueryProvider } from "@/components/providers/query-provider";
import { RouterProvider, createRouter } from "@tanstack/react-router";
import { routeTree } from "./routeTree.gen";

// Set up a Router instance
const router = createRouter({
	routeTree,
	defaultPreload: "intent",
});

// Register things for typesafety
declare module "@tanstack/react-router" {
	interface Register {
		router: typeof router;
	}
}
const App = () => {
	return (
		<QueryProvider>
			<RouterProvider router={router} />
		</QueryProvider>
	);
};

export default App;
