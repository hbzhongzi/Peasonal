import { createBrowserRouter } from 'react-router-dom'
import { RootLayout } from '@/layouts/RootLayout'
import { AdminLayout } from '@/layouts/AdminLayout'
import { StudioLayout } from '@/layouts/StudioLayout'
import { ProtectedRoute } from '@/components/ProtectedRoute'
import { NotFoundPage } from '@/pages/NotFoundPage'
import { AboutPage } from '@/pages/AboutPage'
import { LoginPage } from '@/pages/LoginPage'
import { AdminLoginPage } from '@/pages/AdminLoginPage'
import { RegisterPage } from '@/pages/RegisterPage'
import { AdminProtectedRoute } from '@/components/AdminProtectedRoute'
import { PostListPage } from '@/features/posts/pages/PostListPage'
import { PostDetailPage } from '@/features/posts/pages/PostDetailPage'
import { TagListPage, TagPostsPage } from '@/features/posts/pages/TagPage'
import { ArchivePage, ArchiveMonthPage } from '@/features/posts/pages/ArchivePage'
import { SearchPage } from '@/features/posts/pages/SearchPage'
import { AdminPostListPage } from '@/features/posts/pages/AdminPostListPage'
import { PostEditorPage } from '@/features/posts/pages/PostEditorPage'
import { MediaLibraryPage } from '@/features/media/pages/MediaLibraryPage'
import { AdminCommentsPage } from '@/features/comments/pages/AdminCommentsPage'
import { UsersAdminPage } from '@/features/auth/pages/UsersAdminPage'
import { ProfilePage } from '@/features/auth/pages/ProfilePage'
import { UserBlogPage } from '@/features/users/pages/UserBlogPage'

export const router = createBrowserRouter([
  {
    path: '/',
    element: <RootLayout />,
    children: [
      { index: true, element: <PostListPage /> },
      { path: 'posts/:slug', element: <PostDetailPage /> },
      // 某个用户的博客主页：他的公开资料 + 他发布的文章
      { path: 'blog/:publicID', element: <UserBlogPage /> },
      { path: 'about', element: <AboutPage /> },
      { path: 'tags', element: <TagListPage /> },
      { path: 'tags/:slug', element: <TagPostsPage /> },
      { path: 'archive', element: <ArchivePage /> },
      { path: 'archive/:year/:month', element: <ArchiveMonthPage /> },
      { path: 'search', element: <SearchPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/register',
    element: <RegisterPage />,
  },
  {
    // 「我的博客」：个人博客编辑管理界面。发帖权限(can_publish)用户可进入，
    // 在此写文章、管媒体、审评论——这是用户自己的工作台，不是后台管理。
    path: '/studio',
    element: (
      <ProtectedRoute>
        <StudioLayout />
      </ProtectedRoute>
    ),
    children: [
      { index: true, element: <AdminPostListPage /> },
      { path: 'posts/new', element: <PostEditorPage /> },
      { path: 'posts/:id', element: <PostEditorPage /> },
      { path: 'media', element: <MediaLibraryPage /> },
      { path: 'profile', element: <ProfilePage /> },
    ],
  },
  {
    // 管理员专属登录入口：与普通登录页视觉隔离，登录后校验 is_admin。
    path: '/admin/login',
    element: <AdminLoginPage />,
  },
  {
    // 「后台」：纯管理后台。仅站点管理员(is_admin)可进入，管理用户与权限。
    path: '/admin',
    element: (
      <AdminProtectedRoute>
        <AdminLayout />
      </AdminProtectedRoute>
    ),
    children: [
      { index: true, element: <UsersAdminPage /> },
      // 评论是全站数据（留言者是匿名访客），管理入口放在纯管理后台
      { path: 'comments', element: <AdminCommentsPage /> },
    ],
  },
])
