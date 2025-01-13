QudStack 🫦
===========

🤨 What is this?
================

A Typescript React boilerplate for frontend development that consists of Quddus personal favourite packages.

🔧 Current feature
==================

-	RsBuild and RsPack for buildkit
-	React Compiler
-	Biome for linting
-	Tanstack Router for file-based routing
-	Tanstack Query for data fetching
-	TailwindCSS for styling

Routing Tutorial
----------------

Routing is done via file-based routing thanks to [TanStack Router](https://tanstack.com/router/latest). This means that each route is a file in the `src/routes` directory. The file should export a React component as default.

1.	Make sure the app is running via `p run dev`
2.	Create a new file in `src/routes` named either `routename/index.tsx` or `routename.tsx`. Both are the same.
3.	The file should export a React component as default.
